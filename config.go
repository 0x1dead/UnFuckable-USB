package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const (
	AppName    = "UnFuckable USB"
	AppVersion = "1.0.1"
	AppAuthor  = "0x1dead"
	AppTagline = "Making your data impossible to fuck with"
	AppYear    = "2025"
)

const (
	Argon2Time      = 4
	Argon2Memory    = 1024 * 1024 // 1 GB
	Argon2Threads   = 8
	Argon2KeyLength = 32

	SaltSize       = 32
	NonceSize      = 12
	XNonceSize     = 24
	SessionKeySize = 32
	ChunkSize      = 16 * 1024 * 1024

	WipePasses = 3

	MinDecoyFiles = 50
	MaxDecoyFiles = 200
	MinDecoySize  = 1024
	MaxDecoySize  = 1024 * 1024

	DefaultAutoLockMinutes = 5
	SessionExpiryHours     = 24 * 7

	UIWidth      = 90
	UIHeight     = 24
	ProgressWidth = 50

	ManifestFile = ".sys"
	ExcludeFile  = ".unfuckable.exclude"
)

type Config struct {
	Language        string            `json:"language"`
	Theme           string            `json:"theme"`
	AutoLockMinutes int               `json:"auto_lock_minutes"`
	SecureWipe      bool              `json:"secure_wipe"`
	DoubleEncrypt   bool              `json:"double_encrypt"`
	PanicHotkey     string            `json:"panic_hotkey"`
	PanicEnabled    bool              `json:"panic_enabled"`
	GenerateDecoys  bool              `json:"generate_decoys"`
	DecoyCount      int               `json:"decoy_count"`
	Sessions        map[string]string `json:"sessions"`
	Exclusions      []string          `json:"exclusions"`
	ConfirmActions  bool              `json:"confirm_actions"`
	LastDrive       string            `json:"last_drive"`
}

var AppConfig = &Config{
	Language:        "en",
	Theme:           "default",
	AutoLockMinutes: DefaultAutoLockMinutes,
	SecureWipe:      true,
	DoubleEncrypt:   true,
	PanicHotkey:     "Ctrl+Shift+F12",
	PanicEnabled:    true,
	GenerateDecoys:  true,
	DecoyCount:      100,
	Sessions:        make(map[string]string),
	Exclusions:      []string{},
	ConfirmActions:  true,
	LastDrive:       "",
}

func getConfigDir() string {
	var dir string

	switch runtime.GOOS {
	case "windows":
		if appData := os.Getenv("APPDATA"); appData != "" {
			dir = filepath.Join(appData, "UnFuckableUSB")
		}
	case "darwin":
		if home := os.Getenv("HOME"); home != "" {
			dir = filepath.Join(home, "Library", "Application Support", "UnFuckableUSB")
		}
	default:
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			dir = filepath.Join(xdg, "unfuckable-usb")
		} else if home := os.Getenv("HOME"); home != "" {
			dir = filepath.Join(home, ".config", "unfuckable-usb")
		}
	}

	if dir == "" {
		dir = "."
	}

	os.MkdirAll(dir, 0700)
	return dir
}

func getConfigPath() string {
	return filepath.Join(getConfigDir(), "config.json")
}

func LoadConfig() error {
	data, err := os.ReadFile(getConfigPath())
	if err != nil {
		AppConfig.Language = detectLanguage()
		return SaveConfig()
	}

	if err := json.Unmarshal(data, AppConfig); err != nil {
		return err
	}

	if AppConfig.Sessions == nil {
		AppConfig.Sessions = make(map[string]string)
	}

	return nil
}

func SaveConfig() error {
	data, err := json.MarshalIndent(AppConfig, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(getConfigPath(), data, 0600)
}

func detectLanguage() string {
	lang := os.Getenv("LANG")
	if len(lang) >= 2 {
		switch lang[:2] {
		case "ru":
			return "ru"
		case "uk":
			return "uk"
		}
	}
	return "en"
}

func RandomHex(n int) string {
	b := make([]byte, n/2)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func RandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	return b, err
}

func Now() time.Time {
	return time.Now()
}
