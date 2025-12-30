package main

import (
	"sync"
	"time"
)

type PanicManager struct {
	enabled    bool
	running    bool
	mu         sync.Mutex
	stopChan   chan struct{}
	onTrigger  func()
	lastPanic  time.Time
	panicCount int
}

var Panic = &PanicManager{
	enabled:  true,
	stopChan: make(chan struct{}),
}

// SetCallback sets panic trigger callback
func (p *PanicManager) SetCallback(fn func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onTrigger = fn
}

// Start begins listening for panic hotkey
func (p *PanicManager) Start() {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return
	}
	p.running = true
	p.mu.Unlock()

	go p.listen()
}

// Stop stops panic listener
func (p *PanicManager) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return
	}

	p.running = false
	close(p.stopChan)
	p.stopChan = make(chan struct{})
}

// Trigger manually triggers panic
func (p *PanicManager) Trigger() {
	p.mu.Lock()
	p.lastPanic = time.Now()
	p.panicCount++
	callback := p.onTrigger
	p.mu.Unlock()

	if callback != nil {
		callback()
	}
}

// IsEnabled returns if panic is enabled
func (p *PanicManager) IsEnabled() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.enabled
}

// SetEnabled enables/disables panic
func (p *PanicManager) SetEnabled(enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.enabled = enabled
}

// listen listens for panic hotkey
func (p *PanicManager) listen() {
	if IsGlobalHotkeySupported() {
		// Register global hotkey (works even when minimized)
		RegisterPanicHotkey(func() {
			if p.enabled {
				p.Trigger()
			}
		})
	}
	
	// Wait for stop signal
	<-p.stopChan
	
	if IsGlobalHotkeySupported() {
		UnregisterPanicHotkey()
	}
}

// IsGlobalHotkeyAvailable returns true if global hotkey is supported
func IsGlobalHotkeyAvailable() bool {
	return IsGlobalHotkeySupported()
}

// GetHotkeyStatus returns status string for UI
func GetHotkeyStatus() string {
	if IsGlobalHotkeySupported() {
		return "Ctrl+Shift+F12 (" + T("global") + ")"
	}
	return "F12 (" + T("in_app_only") + ")"
}

// EncryptAllDecrypted encrypts all currently decrypted drives
func EncryptAllDecrypted(progress ProgressFunc) []error {
	devices, err := ScanDevices()
	if err != nil {
		return []error{err}
	}

	var errors []error

	for _, dev := range devices {
		if dev.IsEncrypted {
			continue
		}

		if !Sessions.Has(dev.DriveID) {
			continue
		}

		err := QuickEncrypt(dev.Path, dev.DriveID, progress)
		if err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

// PanicEncrypt is the panic button action
func PanicEncrypt() {
	errors := EncryptAllDecrypted(nil)
	_ = errors // In panic mode we don't care about individual errors
}

// GetPanicStats returns panic statistics
func (p *PanicManager) GetPanicStats() (count int, lastTime time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.panicCount, p.lastPanic
}
