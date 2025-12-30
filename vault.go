package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	DoubleEncrypt bool              `json:"de"` // was double encryption used?
}

type ProgressFunc func(current, total int64, stage string)

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

	totalSize := calcTotalSize(files)

	// Create temp archive
	archivePath := filepath.Join(drivePath, ".tmp_"+RandomHex(8))
	_, err = createArchive(drivePath, archivePath, files, func(c, t int64) {
		if progress != nil {
			progress(c, totalSize*2, T("archiving"))
		}
	})
	if err != nil {
		os.Remove(archivePath)
		return fmt.Errorf("archive failed: %w", err)
	}

	// Read archive
	archiveData, err := os.ReadFile(archivePath)
	os.Remove(archivePath)
	if err != nil {
		return err
	}

	if progress != nil {
		progress(totalSize, totalSize*2, T("encrypting"))
	}

	// Encrypt
	encrypted, err := Encrypt(archiveData, password)
	if err != nil {
		return fmt.Errorf("encryption failed: %w", err)
	}

	// Generate obfuscated file names
	manifest := &VaultManifest{
		Version:       AppVersion,
		Created:       time.Now(),
		Modified:      time.Now(),
		OriginalSize:  totalSize,
		FileCount:     len(files),
		Files:         make(map[string]string),
		DoubleEncrypt: AppConfig.DoubleEncrypt, // save encryption mode
	}

	manifest.Salt, _ = GenerateSalt()

	// Generate decoy files if enabled
	var decoyFiles []string
	if AppConfig.GenerateDecoys {
		decoyFiles = generateDecoyFileNames(AppConfig.DecoyCount)
		manifest.HasDecoy = true
	}

	// Write encrypted vault with obfuscated name
	vaultName := RandomHex(16)
	vaultPath := filepath.Join(drivePath, "."+vaultName)
	if err := os.WriteFile(vaultPath, encrypted, 0644); err != nil {
		return err
	}
	manifest.Files["__vault__"] = vaultName

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

	manifestPath := filepath.Join(drivePath, ManifestFile)
	if err := os.WriteFile(manifestPath, encManifest, 0644); err != nil {
		return err
	}

	// Secure delete original files
	for _, file := range files {
		if AppConfig.SecureWipe {
			SecureDelete(file)
		} else {
			os.Remove(file)
		}
	}

	// Clean empty directories
	cleanEmptyDirs(drivePath)

	// CLEAR session after encryption - drive is now locked!
	Sessions.Clear(driveID)

	if progress != nil {
		progress(totalSize*2, totalSize*2, T("done"))
	}

	return nil
}

func DecryptDrive(drivePath, driveID, password string, progress ProgressFunc) error {
	// Load manifest
	manifest, err := loadManifest(drivePath, password)
	if err != nil {
		return fmt.Errorf("wrong password or corrupted vault")
	}

	// Find vault file
	vaultName, ok := manifest.Files["__vault__"]
	if !ok {
		return fmt.Errorf("vault file not found")
	}

	vaultPath := filepath.Join(drivePath, "."+vaultName)
	encrypted, err := os.ReadFile(vaultPath)
	if err != nil {
		return fmt.Errorf("vault read failed: %w", err)
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

	// Remove vault files
	os.Remove(vaultPath)
	os.Remove(filepath.Join(drivePath, ManifestFile))

	// Remove decoy files
	removeDecoyFiles(drivePath)

	// CREATE session after decryption - now we can quick-encrypt later!
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
	// ChangePassword works on DECRYPTED drive only!
	// We have files visible, and we want to change what password will be used for next encryption
	
	// Just update the session with new password
	// Next time QuickEncrypt is called, it will use the new password
	Sessions.Set(driveID, drivePath, newPassword)

	return nil
}

func EraseVault(drivePath, driveID string) error {
	// Remove all hidden files
	entries, _ := os.ReadDir(drivePath)
	for _, e := range entries {
		if len(e.Name()) > 0 && e.Name()[0] == '.' {
			path := filepath.Join(drivePath, e.Name())
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

func scanFiles(root string, exclusions []string) ([]string, error) {
	var files []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || path == root || info.IsDir() {
			return err
		}

		// Skip hidden files
		if info.Name()[0] == '.' {
			return nil
		}

		// Check exclusions
		relPath, _ := filepath.Rel(root, path)
		for _, excl := range exclusions {
			if matchExclusion(relPath, excl) {
				return nil
			}
		}

		files = append(files, path)
		return nil
	})

	return files, err
}

func matchExclusion(path, pattern string) bool {
	// Normalize path separators
	path = filepath.ToSlash(path)
	pattern = filepath.ToSlash(pattern)

	// Simple pattern matching
	if pattern == path {
		return true
	}

	// Wildcard at end
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		if len(path) >= len(prefix) && path[:len(prefix)] == prefix {
			return true
		}
	}

	// Contains check
	if len(pattern) > 2 && pattern[0] == '*' && pattern[len(pattern)-1] == '*' {
		substr := pattern[1 : len(pattern)-1]
		return containsString(path, substr)
	}

	return false
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func calcTotalSize(files []string) int64 {
	var total int64
	for _, f := range files {
		if info, err := os.Stat(f); err == nil {
			total += info.Size()
		}
	}
	return total
}

func createArchive(baseDir, outputPath string, files []string, progress func(int64, int64)) (int64, error) {
	out, err := os.Create(outputPath)
	if err != nil {
		return 0, err
	}
	defer out.Close()

	gw := gzip.NewWriter(out)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	var processed int64
	total := calcTotalSize(files)

	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			continue
		}

		relPath, _ := filepath.Rel(baseDir, file)
		header.Name = relPath

		tw.WriteHeader(header)

		f, err := os.Open(file)
		if err != nil {
			continue
		}

		io.Copy(tw, f)
		f.Close()

		processed += info.Size()
		if progress != nil {
			progress(processed, total)
		}
	}

	return total, nil
}

func extractArchive(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(destDir, header.Name)

		if header.Typeflag == tar.TypeReg {
			os.MkdirAll(filepath.Dir(target), 0755)
			out, err := os.Create(target)
			if err != nil {
				return err
			}
			io.Copy(out, tr)
			out.Close()
		}
	}

	return nil
}

func cleanEmptyDirs(root string) {
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() || path == root {
			return nil
		}
		entries, _ := os.ReadDir(path)
		if len(entries) == 0 {
			os.Remove(path)
		}
		return nil
	})
}

func loadExclusions(drivePath string) []string {
	exclusions := make([]string, len(AppConfig.Exclusions))
	copy(exclusions, AppConfig.Exclusions)

	// Load from drive
	exclPath := filepath.Join(drivePath, ExcludeFile)
	if data, err := os.ReadFile(exclPath); err == nil {
		lines := splitLines(string(data))
		for _, line := range lines {
			line = trimSpace(line)
			if line != "" && line[0] != '#' {
				exclusions = append(exclusions, line)
			}
		}
	}

	return exclusions
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
