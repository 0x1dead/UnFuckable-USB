package main

import (
	"sync"
	"time"
)

type AutoLock struct {
	enabled      bool
	timeout      time.Duration
	lastActivity time.Time
	mu           sync.Mutex
	stopChan     chan struct{}
	running      bool
	onLock       func()
}

var AutoLocker = &AutoLock{
	enabled:      true,
	timeout:      time.Duration(DefaultAutoLockMinutes) * time.Minute,
	lastActivity: time.Now(),
	stopChan:     make(chan struct{}),
}

// SetTimeout sets auto-lock timeout
func (al *AutoLock) SetTimeout(minutes int) {
	al.mu.Lock()
	defer al.mu.Unlock()
	al.timeout = time.Duration(minutes) * time.Minute
}

// SetEnabled enables/disables auto-lock
func (al *AutoLock) SetEnabled(enabled bool) {
	al.mu.Lock()
	defer al.mu.Unlock()
	al.enabled = enabled
}

// SetCallback sets lock callback
func (al *AutoLock) SetCallback(fn func()) {
	al.mu.Lock()
	defer al.mu.Unlock()
	al.onLock = fn
}

// Touch resets activity timer
func (al *AutoLock) Touch() {
	al.mu.Lock()
	defer al.mu.Unlock()
	al.lastActivity = time.Now()
}

// Start begins auto-lock monitoring
func (al *AutoLock) Start() {
	al.mu.Lock()
	if al.running {
		al.mu.Unlock()
		return
	}
	al.running = true
	al.mu.Unlock()

	go al.monitor()
}

// Stop stops auto-lock monitoring
func (al *AutoLock) Stop() {
	al.mu.Lock()
	defer al.mu.Unlock()

	if !al.running {
		return
	}

	al.running = false
	close(al.stopChan)
	al.stopChan = make(chan struct{})
}

// monitor checks for inactivity
func (al *AutoLock) monitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-al.stopChan:
			return
		case <-ticker.C:
			al.check()
		}
	}
}

// check checks if timeout reached
func (al *AutoLock) check() {
	al.mu.Lock()
	defer al.mu.Unlock()

	if !al.enabled {
		return
	}

	if time.Since(al.lastActivity) > al.timeout {
		if al.onLock != nil {
			al.onLock()
		}
		al.lastActivity = time.Now()
	}
}

// TimeRemaining returns time until auto-lock
func (al *AutoLock) TimeRemaining() time.Duration {
	al.mu.Lock()
	defer al.mu.Unlock()

	if !al.enabled {
		return 0
	}

	elapsed := time.Since(al.lastActivity)
	if elapsed >= al.timeout {
		return 0
	}

	return al.timeout - elapsed
}

// IsEnabled returns if auto-lock is enabled
func (al *AutoLock) IsEnabled() bool {
	al.mu.Lock()
	defer al.mu.Unlock()
	return al.enabled
}
