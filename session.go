package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"sync"
	"time"
)

type Session struct {
	Password    []byte    `json:"-"`
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

// FIX: Кэширование machineKey
var (
	cachedMachineKey []byte
	machineKeyOnce   sync.Once
)

func machineKey() []byte {
	machineKeyOnce.Do(func() {
		hostname, _ := os.Hostname()
		if hostname == "" {
			hostname = "local"
		}
		
		entropy := hostname + getConfigDir()
		
		h := HMAC256([]byte(entropy), []byte("unfuckable-machine-v1"))
		cachedMachineKey = make([]byte, 32)
		copy(cachedMachineKey, h)
	})
	
	return cachedMachineKey
}

func (sm *SessionManager) Set(driveID, drivePath, password string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	encPw, err := encryptSessionPassword(password)
	if err != nil {
		return err
	}

	session := &Session{
		Password:    []byte(password),
		DriveID:     driveID,
		DrivePath:   drivePath,
		CreatedAt:   time.Now(),
		LastUsed:    time.Now(),
		EncryptedPw: encPw,
	}

	sm.sessions[driveID] = session

	AppConfig.Sessions[driveID] = encPw
	SaveConfig()

	return nil
}

// FIX: Исправлена race condition с LastUsed
func (sm *SessionManager) Get(driveID string) (string, bool) {
	// Сначала проверяем с read lock
	sm.mu.RLock()
	if s, ok := sm.sessions[driveID]; ok {
		sm.mu.RUnlock()
		
		// Обновляем LastUsed с write lock
		sm.mu.Lock()
		s.LastUsed = time.Now()
		sm.mu.Unlock()
		
		return string(s.Password), true
	}
	sm.mu.RUnlock()

	// Нужно создать новую сессию - получаем write lock
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	// Double-check после получения write lock
	if s, ok := sm.sessions[driveID]; ok {
		s.LastUsed = time.Now()
		return string(s.Password), true
	}

	// Проверяем в AppConfig
	if encPw, ok := AppConfig.Sessions[driveID]; ok {
		password, err := decryptSessionPassword(encPw)
		if err != nil {
			return "", false
		}

		// Создаем новую сессию в памяти
		sm.sessions[driveID] = &Session{
			Password:    []byte(password),
			DriveID:     driveID,
			LastUsed:    time.Now(),
			EncryptedPw: encPw,
		}

		return password, true
	}

	return "", false
}

func (sm *SessionManager) Has(driveID string) bool {
	_, ok := sm.Get(driveID)
	return ok
}

func (sm *SessionManager) Clear(driveID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if s, ok := sm.sessions[driveID]; ok {
		SecureZero(s.Password)
	}

	delete(sm.sessions, driveID)
	delete(AppConfig.Sessions, driveID)
	SaveConfig()
}

func (sm *SessionManager) ClearAll() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for _, s := range sm.sessions {
		SecureZero(s.Password)
	}

	sm.sessions = make(map[string]*Session)
	AppConfig.Sessions = make(map[string]string)
	SaveConfig()
}

func (sm *SessionManager) GetAll() map[string]*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make(map[string]*Session)
	for k, v := range sm.sessions {
		result[k] = v
	}
	return result
}

func (sm *SessionManager) LoadFromConfig() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for driveID, encPw := range AppConfig.Sessions {
		password, err := decryptSessionPassword(encPw)
		if err != nil {
			continue
		}

		sm.sessions[driveID] = &Session{
			Password:    []byte(password),
			DriveID:     driveID,
			EncryptedPw: encPw,
			LastUsed:    time.Now(),
		}
	}
}

func encryptSessionPassword(password string) (string, error) {
	key := deriveSessionKey()
	
	salt := make([]byte, 16)
	rand.Read(salt)
	
	data := append(salt, []byte(password)...)
	
	encrypted, err := EncryptAESGCM(data, key)
	if err != nil {
		SecureZero(key)
		return "", err
	}
	
	SecureZero(key)
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

func decryptSessionPassword(encrypted string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}

	key := deriveSessionKey()
	decrypted, err := DecryptAESGCM(data, key)
	if err != nil {
		SecureZero(key)
		return "", err
	}
	
	SecureZero(key)
	
	if len(decrypted) < 16 {
		SecureZero(decrypted)
		return "", ErrInvalidData
	}
	
	password := string(decrypted[16:])
	SecureZero(decrypted)
	
	return password, nil
}

func deriveSessionKey() []byte {
	mKey := machineKey()
	// FIX: machineKey теперь кэшируется, безопасно использовать
	// НЕ вызываем SecureZero на cached ключ!
	derived := DeriveKeyFast("session_unfuckable_v2", mKey)
	return derived
}

type SessionInfo struct {
	DriveID   string
	DrivePath string
	LastUsed  time.Time
	Active    bool
}

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

func (sm *SessionManager) Export() ([]byte, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	export := make(map[string]string)
	for id, s := range sm.sessions {
		export[id] = s.EncryptedPw
	}
	
	return json.Marshal(export)
}