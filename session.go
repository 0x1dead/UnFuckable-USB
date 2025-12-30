package main

import (
	"encoding/base64"
	"encoding/json"
	"sync"
	"time"
)

type Session struct {
	Password    string    `json:"-"`
	DriveID     string    `json:"drive_id"`
	DrivePath   string    `json:"drive_path"`
	CreatedAt   time.Time `json:"created_at"`
	LastUsed    time.Time `json:"last_used"`
	EncryptedPw string    `json:"encrypted_pw"`
}

type SessionManager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

var Sessions = &SessionManager{
	sessions: make(map[string]*Session),
}

// machineKey returns unique key for this machine
func machineKey() []byte {
	// Combine hostname + config dir as machine identifier
	hostname, _ := getHostname()
	return []byte(hostname + getConfigDir())
}

func getHostname() (string, error) {
	// Simple implementation
	return "local", nil
}

// Set creates or updates session for drive
func (sm *SessionManager) Set(driveID, drivePath, password string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Encrypt password for storage
	encPw, err := encryptSessionPassword(password)
	if err != nil {
		return err
	}

	session := &Session{
		Password:    password,
		DriveID:     driveID,
		DrivePath:   drivePath,
		CreatedAt:   time.Now(),
		LastUsed:    time.Now(),
		EncryptedPw: encPw,
	}

	sm.sessions[driveID] = session

	// Save to config
	AppConfig.Sessions[driveID] = encPw
	SaveConfig()

	return nil
}

// Get returns password for drive if session exists
func (sm *SessionManager) Get(driveID string) (string, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Check memory first
	if s, ok := sm.sessions[driveID]; ok {
		s.LastUsed = time.Now()
		return s.Password, true
	}

	// Check config
	if encPw, ok := AppConfig.Sessions[driveID]; ok {
		password, err := decryptSessionPassword(encPw)
		if err != nil {
			return "", false
		}

		// Cache in memory
		sm.sessions[driveID] = &Session{
			Password:    password,
			DriveID:     driveID,
			LastUsed:    time.Now(),
			EncryptedPw: encPw,
		}

		return password, true
	}

	return "", false
}

// Has checks if session exists
func (sm *SessionManager) Has(driveID string) bool {
	_, ok := sm.Get(driveID)
	return ok
}

// Clear removes session for drive
func (sm *SessionManager) Clear(driveID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.sessions, driveID)
	delete(AppConfig.Sessions, driveID)
	SaveConfig()
}

// ClearAll removes all sessions
func (sm *SessionManager) ClearAll() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.sessions = make(map[string]*Session)
	AppConfig.Sessions = make(map[string]string)
	SaveConfig()
}

// GetAll returns all active sessions
func (sm *SessionManager) GetAll() map[string]*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make(map[string]*Session)
	for k, v := range sm.sessions {
		result[k] = v
	}
	return result
}

// LoadFromConfig loads sessions from config
func (sm *SessionManager) LoadFromConfig() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for driveID, encPw := range AppConfig.Sessions {
		password, err := decryptSessionPassword(encPw)
		if err != nil {
			continue
		}

		sm.sessions[driveID] = &Session{
			Password:    password,
			DriveID:     driveID,
			EncryptedPw: encPw,
			LastUsed:    time.Now(),
		}
	}
}

// encryptSessionPassword encrypts password for storage
func encryptSessionPassword(password string) (string, error) {
	key := deriveSessionKey()
	encrypted, err := EncryptAESGCM([]byte(password), key)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// decryptSessionPassword decrypts stored password
func decryptSessionPassword(encrypted string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}

	key := deriveSessionKey()
	decrypted, err := DecryptAESGCM(data, key)
	if err != nil {
		return "", err
	}

	return string(decrypted), nil
}

// deriveSessionKey creates key for session encryption
func deriveSessionKey() []byte {
	return DeriveKeyFast("session_"+string(machineKey()), machineKey())
}

// SessionInfo for UI display
type SessionInfo struct {
	DriveID   string
	DrivePath string
	LastUsed  time.Time
	Active    bool
}

// GetSessionsInfo returns info for all sessions
func (sm *SessionManager) GetSessionsInfo() []SessionInfo {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var result []SessionInfo
	for id, s := range sm.sessions {
		result = append(result, SessionInfo{
			DriveID:   id,
			DrivePath: s.DrivePath,
			LastUsed:  s.LastUsed,
			Active:    true,
		})
	}
	return result
}

// Export sessions to JSON (for backup)
func (sm *SessionManager) Export() ([]byte, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return json.Marshal(sm.sessions)
}
