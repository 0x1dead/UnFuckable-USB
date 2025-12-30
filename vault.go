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

type VaultManifest struct {
	Version       string            `json:"v"`
	Created       time.Time         `json:"c"`
	Modified      time.Time         `json:"m"`
	OriginalSize  int64             `json:"os"`
	FileCount     int               `json:"fc"`
	Files         map[string]string `json:"f"` // real_name -> obfuscated_name
	Salt          []byte            `json:"s"`
	HasDecoy      bool              `json:"d"`
	DoubleEncrypt bool              `json:"de"`

	// Chunk info
	UseChunks   bool     `json:"uc"`
	ChunkNames  []string `json:"cn"` // ordered list of chunk file names
	ChunkSizes  []int64  `json:"cs"` // size of each chunk
	TotalChunks int      `json:"tc"`
}

type ProgressFunc func(current, total int64, stage string)

// Random extensions for chunks
var chunkExtensions = []string{
	".tmp", ".bak", ".old", ".log", ".dat", ".bin", ".cache",
	".db", ".idx", ".swp", ".temp", "~", ".part", ".download",
	".crdownload", ".partial", ".!ut", ".bc!", ".aria2",
}

func EncryptDrive(drivePath, driveID, password string, progress ProgressFunc) error {
	// Load exclusions
	exclusions := loadExclusions(drivePath)

	// Scan files
	files, err := scanFiles(drivePath, exclusions)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no files to encrypt")
	}

	// Calculate total size
	var totalSize int64
	for _, f := range files {
		totalSize += f.Size()
	}

	if progress != nil {
		progress(0, totalSize, T("compressing"))
	}

	// Create archive
	archivePath := filepath.Join(drivePath, ".tmp_"+RandomHex(8))
	if err := createArchive(files, drivePath, archivePath, progress, totalSize); err != nil {
		return fmt.Errorf("archive failed: %w", err)
	}
	defer os.Remove(archivePath)

	// Read archive
	archiveData, err := os.ReadFile(archivePath)
	if err != nil {
		return fmt.Errorf("read archive failed: %w", err)
	}

	if progress != nil {
		progress(totalSize/2, totalSize, T("encrypting"))
	}

	// Encrypt archive (inner layer)
	encrypted, err := Encrypt(archiveData, password)
	if err != nil {
		return fmt.Errorf("encryption failed: %w", err)
	}

	// Generate manifest
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

	// Generate decoy files if enabled
	var decoyFiles []string
	if AppConfig.GenerateDecoys {
		decoyFiles = generateDecoyFileNames(AppConfig.DecoyCount)
		manifest.HasDecoy = true
	}

	// Write encrypted data
	if AppConfig.UseChunks {
		// Split into chunks with random sizes
		if err := writeChunks(drivePath, encrypted, manifest); err != nil {
			return fmt.Errorf("write chunks failed: %w", err)
		}
	} else {
		// Single vault file
		vaultName := RandomHex(16)
		vaultPath := filepath.Join(drivePath, "."+vaultName)
		if err := os.WriteFile(vaultPath, encrypted, 0644); err != nil {
			return err
		}
		manifest.Files["__vault__"] = vaultName
	}

	// Write decoy files
	for _, name := range decoyFiles {
		decoyPath := filepath.Join(drivePath, "."+name)
		decoyData := generateDecoyData()
		os.WriteFile(decoyPath, decoyData, 0644)
	}

	// Encrypt and save manifest
	manifestData, _ := json.Marshal(manifest)
	encManifest, err := Encrypt(manifestData, password)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(drivePath, ManifestFile), encManifest, 0644); err != nil {
		return err
	}

	// Delete original files
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

	// Remove empty directories
	removeEmptyDirs(drivePath)

	// Clear session after encryption
	Sessions.Clear(driveID)

	if progress != nil {
		progress(totalSize, totalSize, T("done"))
	}

	return nil
}

func writeChunks(drivePath string, data []byte, manifest *VaultManifest) error {
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

	offset := 0
	chunkIndex := 0
	dataLen := len(data)

	for offset < dataLen {
		// Calculate chunk size with variance
		thisChunkSize := chunkSize
		if variance > 0 {
			// Random variance: -variance% to +variance%
			varianceRange := int64(chunkSize * variance / 100)
			if varianceRange > 0 {
				randVariance, _ := rand.Int(rand.Reader, big.NewInt(varianceRange*2))
				thisChunkSize = chunkSize - int(varianceRange) + int(randVariance.Int64())
			}
		}

		// Clamp to remaining data
		if offset+thisChunkSize > dataLen {
			thisChunkSize = dataLen - offset
		}

		// Generate random chunk name
		chunkName := generateRandomChunkName()
		chunkPath := filepath.Join(drivePath, chunkName)

		// Write chunk
		chunkData := data[offset : offset+thisChunkSize]
		if err := os.WriteFile(chunkPath, chunkData, 0644); err != nil {
			return err
		}

		manifest.ChunkNames = append(manifest.ChunkNames, chunkName)
		manifest.ChunkSizes = append(manifest.ChunkSizes, int64(thisChunkSize))

		offset += thisChunkSize
		chunkIndex++
	}

	manifest.TotalChunks = chunkIndex
	return nil
}

func generateRandomChunkName() string {
	// Random prefix (looks like temp/system file)
	prefixes := []string{
		"~$", "~", ".", ".~", "$", "._",
	}

	// Random middle part
	middles := []string{
		RandomHex(8),
		RandomHex(12),
		RandomHex(16),
		fmt.Sprintf("%d", time.Now().UnixNano()%1000000),
	}

	// Random extension
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

func DecryptDrive(drivePath, driveID, password string, progress ProgressFunc) error {
	// Load manifest
	manifest, err := loadManifest(drivePath, password)
	if err != nil {
		return fmt.Errorf("wrong password or corrupted vault")
	}

	var encrypted []byte

	if manifest.UseChunks && len(manifest.ChunkNames) > 0 {
		// Read and combine chunks
		if progress != nil {
			progress(0, manifest.OriginalSize, T("reading_chunks"))
		}

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
	} else {
		// Single vault file
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

	// Decrypt
	decrypted, err := Decrypt(encrypted, password)
	if err != nil {
		return ErrDecryptFailed
	}

	// Write temp archive
	archivePath := filepath.Join(drivePath, ".tmp_"+RandomHex(8))
	if err := os.WriteFile(archivePath, decrypted, 0644); err != nil {
		return err
	}

	if progress != nil {
		progress(manifest.OriginalSize/2, manifest.OriginalSize, T("extracting"))
	}

	// Extract
	if err := extractArchive(archivePath, drivePath); err != nil {
		os.Remove(archivePath)
		return fmt.Errorf("extract failed: %w", err)
	}
	os.Remove(archivePath)

	// Remove chunk files
	if manifest.UseChunks {
		for _, chunkName := range manifest.ChunkNames {
			os.Remove(filepath.Join(drivePath, chunkName))
		}
	} else {
		// Remove single vault file
		if vaultName, ok := manifest.Files["__vault__"]; ok {
			os.Remove(filepath.Join(drivePath, "."+vaultName))
		}
	}

	// Remove manifest
	os.Remove(filepath.Join(drivePath, ManifestFile))

	// Remove decoy files
	removeDecoyFiles(drivePath)

	// CREATE session after decryption
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
	// Remove all hidden files
	entries, _ := os.ReadDir(drivePath)
	for _, e := range entries {
		name := e.Name()
		// Remove hidden files and chunk-like files
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

		// Skip hidden files
		if len(relPath) > 0 && (relPath[0] == '.' || relPath[0] == '~' || relPath[0] == '$') {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Check exclusions
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

		// Ensure parent dir exists
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

	// Sort by depth (deepest first)
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

	// Global exclusions from config
	exclusions = append(exclusions, AppConfig.Exclusions...)

	// Per-drive exclusions
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
