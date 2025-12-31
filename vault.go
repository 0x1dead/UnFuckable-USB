package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type ChunkInfo struct {
	Name string `json:"n"`
	Size int64  `json:"s"`
	HMAC []byte `json:"h"`
}

type VaultManifest struct {
	Version       string            `json:"v"`
	Created       time.Time         `json:"c"`
	Modified      time.Time         `json:"m"`
	OriginalSize  int64             `json:"os"`
	FileCount     int               `json:"fc"`
	Files         map[string]string `json:"f"`
	Salt          []byte            `json:"s"`
	HasDecoy      bool              `json:"d"`
	DoubleEncrypt bool              `json:"de"`

	UseChunks   bool        `json:"uc"`
	Chunks      []ChunkInfo `json:"cks"`
	TotalChunks int         `json:"tc"`
	
	ChunkNames  []string `json:"cn,omitempty"`
	ChunkSizes  []int64  `json:"cs,omitempty"`
}

type ProgressFunc func(current, total int64, stage string)

var chunkExtensions = []string{
	".tmp", ".bak", ".old", ".log", ".dat", ".bin", ".cache",
	".db", ".idx", ".swp", ".temp", "~", ".part", ".download",
	".crdownload", ".partial", ".!ut", ".bc!", ".aria2",
}

func EncryptDrive(drivePath, driveID, password string, progress ProgressFunc) error {
	exclusions := loadExclusions(drivePath)

	files, err := scanFiles(drivePath, exclusions)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no files to encrypt")
	}

	var totalSize int64
	for _, f := range files {
		totalSize += f.Size()
	}

	if progress != nil {
		progress(0, totalSize, T("compressing"))
	}

	archivePath := filepath.Join(drivePath, ".tmp_"+RandomHex(8))
	if err := createArchive(files, drivePath, archivePath, progress, totalSize); err != nil {
		return fmt.Errorf("archive failed: %w", err)
	}
	defer os.Remove(archivePath)

	archiveData, err := os.ReadFile(archivePath)
	if err != nil {
		return fmt.Errorf("read archive failed: %w", err)
	}

	if progress != nil {
		progress(totalSize/2, totalSize, T("encrypting"))
	}

	encrypted, err := Encrypt(archiveData, password)
	if err != nil {
		return fmt.Errorf("encryption failed: %w", err)
	}

	manifest := &VaultManifest{
		Version:       AppVersion,
		Created:       time.Now(),
		Modified:      time.Now(),
		OriginalSize:  totalSize,
		FileCount:     len(files),
		Files:         make(map[string]string),
		DoubleEncrypt: AppConfig.DoubleEncrypt,
		UseChunks:     AppConfig.UseChunks,
	}

	manifest.Salt, _ = GenerateSalt()

	var decoyFiles []string
	if AppConfig.GenerateDecoys {
		decoyFiles = generateDecoyFileNames(AppConfig.DecoyCount)
		manifest.HasDecoy = true
	}

	if AppConfig.UseChunks {
		if err := writeChunks(drivePath, encrypted, manifest, password); err != nil {
			return fmt.Errorf("write chunks failed: %w", err)
		}
	} else {
		vaultName := RandomHex(16)
		vaultPath := filepath.Join(drivePath, "."+vaultName)
		if err := os.WriteFile(vaultPath, encrypted, 0644); err != nil {
			return err
		}
		manifest.Files["__vault__"] = vaultName
	}

	for _, name := range decoyFiles {
		decoyPath := filepath.Join(drivePath, "."+name)
		decoyData := generateDecoyData()
		os.WriteFile(decoyPath, decoyData, 0644)
	}

	manifestData, _ := json.Marshal(manifest)
	encManifest, err := Encrypt(manifestData, password)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(drivePath, ManifestFile), encManifest, 0644); err != nil {
		return err
	}

	if progress != nil {
		progress(totalSize*3/4, totalSize, T("wiping"))
	}

	for _, f := range files {
		path := filepath.Join(drivePath, f.Name())
		if AppConfig.SecureWipe {
			SecureDelete(path)
		} else {
			os.Remove(path)
		}
	}

	removeEmptyDirs(drivePath)
	Sessions.Clear(driveID)

	if progress != nil {
		progress(totalSize, totalSize, T("done"))
	}

	return nil
}

func writeChunks(drivePath string, data []byte, manifest *VaultManifest, password string) error {
	chunkSize := AppConfig.ChunkSizeMB * 1024 * 1024
	if chunkSize < MinChunkSize {
		chunkSize = MinChunkSize
	}
	if chunkSize > MaxChunkSize {
		chunkSize = MaxChunkSize
	}

	variance := AppConfig.ChunkVariance
	if variance < 0 {
		variance = 0
	}
	if variance > 100 {
		variance = 100
	}

	hmacKey := DeriveKeyFast(password+"_hmac", []byte("chunk_integrity"))
	defer SecureZero(hmacKey)

	offset := 0
	chunkIndex := 0
	dataLen := len(data)

	for offset < dataLen {
		thisChunkSize := chunkSize
		if variance > 0 {
			varianceRange := int64(chunkSize * variance / 100)
			if varianceRange > 0 {
				randVariance, _ := rand.Int(rand.Reader, big.NewInt(varianceRange*2))
				thisChunkSize = chunkSize - int(varianceRange) + int(randVariance.Int64())
			}
		}

		if offset+thisChunkSize > dataLen {
			thisChunkSize = dataLen - offset
		}

		chunkName := generateRandomChunkName()
		chunkPath := filepath.Join(drivePath, chunkName)

		chunkData := data[offset : offset+thisChunkSize]
		if err := os.WriteFile(chunkPath, chunkData, 0644); err != nil {
			return err
		}

		chunkHMAC := HMAC256(chunkData, hmacKey)

		manifest.Chunks = append(manifest.Chunks, ChunkInfo{
			Name: chunkName,
			Size: int64(thisChunkSize),
			HMAC: chunkHMAC,
		})

		offset += thisChunkSize
		chunkIndex++
	}

	manifest.TotalChunks = chunkIndex
	return nil
}

func generateRandomChunkName() string {
	prefixes := []string{
		"~$", "~", ".", ".~", "$", "._",
	}

	middles := []string{
		RandomHex(8),
		RandomHex(12),
		RandomHex(16),
		fmt.Sprintf("%d", time.Now().UnixNano()%1000000),
	}

	ext := chunkExtensions[randomIntN(len(chunkExtensions))]

	prefix := prefixes[randomIntN(len(prefixes))]
	middle := middles[randomIntN(len(middles))]

	return prefix + middle + ext
}

func randomIntN(max int) int {
	if max <= 0 {
		return 0
	}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))
	return int(n.Int64())
}

// FIX: Оптимизирована производительность с предварительной аллокацией
func DecryptDrive(drivePath, driveID, password string, progress ProgressFunc) error {
	manifest, err := loadManifest(drivePath, password)
	if err != nil {
		return fmt.Errorf("wrong password or corrupted vault")
	}

	var encrypted []byte

	if manifest.UseChunks && (len(manifest.Chunks) > 0 || len(manifest.ChunkNames) > 0) {
		if progress != nil {
			progress(0, manifest.OriginalSize, T("reading_chunks"))
		}

		hmacKey := DeriveKeyFast(password+"_hmac", []byte("chunk_integrity"))
		defer SecureZero(hmacKey)

		// FIX: Предварительная аллокация для избежания реаллокаций
		var totalSize int64
		if len(manifest.Chunks) > 0 {
			for _, chunk := range manifest.Chunks {
				totalSize += chunk.Size
			}
		} else {
			for _, size := range manifest.ChunkSizes {
				totalSize += size
			}
		}
		
		// FIX: Аллокация заранее
		encrypted = make([]byte, 0, totalSize)

		if len(manifest.Chunks) > 0 {
			for i, chunk := range manifest.Chunks {
				chunkPath := filepath.Join(drivePath, chunk.Name)
				chunkData, err := os.ReadFile(chunkPath)
				if err != nil {
					return fmt.Errorf("chunk read failed: %s: %w", chunk.Name, err)
				}

				if !VerifyHMAC(chunkData, chunk.HMAC, hmacKey) {
					return fmt.Errorf("chunk integrity check failed: %s", chunk.Name)
				}

				encrypted = append(encrypted, chunkData...)

				if progress != nil {
					progress(int64(i+1)*manifest.OriginalSize/int64(len(manifest.Chunks)),
						manifest.OriginalSize, T("reading_chunks"))
				}
			}
		} else {
			// Legacy format
			for i, chunkName := range manifest.ChunkNames {
				chunkPath := filepath.Join(drivePath, chunkName)
				chunkData, err := os.ReadFile(chunkPath)
				if err != nil {
					return fmt.Errorf("chunk read failed: %s: %w", chunkName, err)
				}
				encrypted = append(encrypted, chunkData...)

				if progress != nil {
					progress(int64(i+1)*manifest.OriginalSize/int64(len(manifest.ChunkNames)),
						manifest.OriginalSize, T("reading_chunks"))
				}
			}
		}
	} else {
		vaultName, ok := manifest.Files["__vault__"]
		if !ok {
			return fmt.Errorf("vault file not found")
		}

		vaultPath := filepath.Join(drivePath, "."+vaultName)
		encrypted, err = os.ReadFile(vaultPath)
		if err != nil {
			return fmt.Errorf("vault read failed: %w", err)
		}
	}

	if progress != nil {
		progress(0, manifest.OriginalSize, T("decrypting"))
	}

	decrypted, err := Decrypt(encrypted, password)
	if err != nil {
		return ErrDecryptFailed
	}

	archivePath := filepath.Join(drivePath, ".tmp_"+RandomHex(8))
	if err := os.WriteFile(archivePath, decrypted, 0644); err != nil {
		return err
	}

	if progress != nil {
		progress(manifest.OriginalSize/2, manifest.OriginalSize, T("extracting"))
	}

	if err := extractArchive(archivePath, drivePath); err != nil {
		os.Remove(archivePath)
		return fmt.Errorf("extract failed: %w", err)
	}
	os.Remove(archivePath)

	if manifest.UseChunks {
		if len(manifest.Chunks) > 0 {
			for _, chunk := range manifest.Chunks {
				os.Remove(filepath.Join(drivePath, chunk.Name))
			}
		} else {
			for _, chunkName := range manifest.ChunkNames {
				os.Remove(filepath.Join(drivePath, chunkName))
			}
		}
	} else {
		if vaultName, ok := manifest.Files["__vault__"]; ok {
			os.Remove(filepath.Join(drivePath, "."+vaultName))
		}
	}

	os.Remove(filepath.Join(drivePath, ManifestFile))
	removeDecoyFiles(drivePath)

	Sessions.Set(driveID, drivePath, password)

	if progress != nil {
		progress(manifest.OriginalSize, manifest.OriginalSize, T("done"))
	}

	return nil
}

func QuickEncrypt(drivePath, driveID string, progress ProgressFunc) error {
	password, ok := Sessions.Get(driveID)
	if !ok {
		return fmt.Errorf("no active session")
	}

	return EncryptDrive(drivePath, driveID, password, progress)
}

func ChangePassword(drivePath, driveID, oldPassword, newPassword string) error {
	Sessions.Set(driveID, drivePath, newPassword)
	return nil
}

func EraseVault(drivePath, driveID string) error {
	entries, _ := os.ReadDir(drivePath)
	for _, e := range entries {
		name := e.Name()
		if len(name) > 0 && (name[0] == '.' || name[0] == '~' || name[0] == '$' || name[0] == '_') {
			path := filepath.Join(drivePath, name)
			if AppConfig.SecureWipe {
				SecureDelete(path)
			} else {
				os.Remove(path)
			}
		}
	}

	Sessions.Clear(driveID)
	return nil
}

func GetVaultInfo(drivePath, password string) (*VaultManifest, error) {
	return loadManifest(drivePath, password)
}

func loadManifest(drivePath, password string) (*VaultManifest, error) {
	manifestPath := filepath.Join(drivePath, ManifestFile)
	encrypted, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	decrypted, err := Decrypt(encrypted, password)
	if err != nil {
		return nil, err
	}

	var manifest VaultManifest
	if err := json.Unmarshal(decrypted, &manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

func scanFiles(root string, exclusions []string) ([]os.FileInfo, error) {
	var files []os.FileInfo

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if path == root {
			return nil
		}

		relPath, _ := filepath.Rel(root, path)

		if len(relPath) > 0 && (relPath[0] == '.' || relPath[0] == '~' || relPath[0] == '$') {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		for _, excl := range exclusions {
			if matched, _ := filepath.Match(excl, info.Name()); matched {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.Contains(relPath, excl) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if !info.IsDir() {
			files = append(files, &fileInfoWithPath{info, relPath})
		}

		return nil
	})

	return files, err
}

type fileInfoWithPath struct {
	os.FileInfo
	path string
}

func (f *fileInfoWithPath) Name() string {
	return f.path
}

func createArchive(files []os.FileInfo, root, archivePath string, progress ProgressFunc, totalSize int64) error {
	file, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	var processed int64

	for _, f := range files {
		filePath := filepath.Join(root, f.Name())

		header, err := tar.FileInfoHeader(f, "")
		if err != nil {
			continue
		}

		header.Name = f.Name()

		if err := tarWriter.WriteHeader(header); err != nil {
			continue
		}

		if !f.IsDir() {
			file, err := os.Open(filePath)
			if err != nil {
				continue
			}

			_, err = io.Copy(tarWriter, file)
			file.Close()

			if err != nil {
				continue
			}

			processed += f.Size()
			if progress != nil {
				progress(processed/2, totalSize, T("compressing"))
			}
		}
	}

	return nil
}

func extractArchive(archivePath, destPath string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		targetPath := filepath.Join(destPath, header.Name)

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(targetPath, os.FileMode(header.Mode))

		case tar.TypeReg:
			outFile, err := os.Create(targetPath)
			if err != nil {
				continue
			}

			_, err = io.Copy(outFile, tarReader)
			outFile.Close()

			if err != nil {
				continue
			}

			os.Chmod(targetPath, os.FileMode(header.Mode))
		}
	}

	return nil
}

func removeEmptyDirs(root string) {
	var dirs []string

	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || path == root {
			return nil
		}
		if info.IsDir() {
			dirs = append(dirs, path)
		}
		return nil
	})

	sort.Slice(dirs, func(i, j int) bool {
		return strings.Count(dirs[i], string(os.PathSeparator)) > strings.Count(dirs[j], string(os.PathSeparator))
	})

	for _, dir := range dirs {
		entries, _ := os.ReadDir(dir)
		if len(entries) == 0 {
			os.Remove(dir)
		}
	}
}

func loadExclusions(drivePath string) []string {
	var exclusions []string

	exclusions = append(exclusions, AppConfig.Exclusions...)

	excludeFile := filepath.Join(drivePath, ExcludeFile)
	data, err := os.ReadFile(excludeFile)
	if err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				exclusions = append(exclusions, line)
			}
		}
	}

	return exclusions
}