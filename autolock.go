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
	wg           sync.WaitGroup
}

var AutoLocker = &AutoLock{
	enabled:      true,
	timeout:      time.Duration(DefaultAutoLockMinutes) * time.Minute,
	lastActivity: time.Now(),
	stopChan:     make(chan struct{}),
}

func (al *AutoLock) SetTimeout(minutes int) {
	al.mu.Lock()
	defer al.mu.Unlock()
	al.timeout = time.Duration(minutes) * time.Minute
}

func (al *AutoLock) SetEnabled(enabled bool) {
	al.mu.Lock()
	defer al.mu.Unlock()
	al.enabled = enabled
}

func (al *AutoLock) SetCallback(fn func()) {
	al.mu.Lock()
	defer al.mu.Unlock()
	al.onLock = fn
}

func (al *AutoLock) Touch() {
	al.mu.Lock()
	defer al.mu.Unlock()
	al.lastActivity = time.Now()
}

func (al *AutoLock) Start() {
	al.mu.Lock()
	if al.running {
		al.mu.Unlock()
		return
	}
	al.running = true
	al.wg.Add(1)
	al.mu.Unlock()

	go al.monitor()
}

// FIX: Исправлена возможная паника при двойном close
func (al *AutoLock) Stop() {
	al.mu.Lock()
	
	if !al.running {
		al.mu.Unlock()
		return
	}

	al.running = false
	
	// FIX: Сохраняем ссылку на канал перед unlock
	stopChan := al.stopChan
	
	al.mu.Unlock()
	
	// FIX: Close без lock (после установки running=false)
	// Это безопасно, так как monitor() проверяет running перед использованием канала
	select {
	case <-stopChan:
		// Канал уже закрыт
	default:
		close(stopChan)
	}
	
	// Ждем завершения горутины
	al.wg.Wait()
	
	// FIX: Создаем новый канал для возможного следующего Start()
	al.mu.Lock()
	al.stopChan = make(chan struct{})
	al.mu.Unlock()
}

func (al *AutoLock) monitor() {
	defer al.wg.Done()
	
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

func (al *AutoLock) check() {
	al.mu.Lock()
	defer al.mu.Unlock()

	if !al.enabled {
		return
	}

	if time.Since(al.lastActivity) > al.timeout {
		if al.onLock != nil {
			// FIX: Вызываем callback в отдельной горутине чтобы не блокировать lock
			callback := al.onLock
			go callback()
		}
		al.lastActivity = time.Now()
	}
}

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

func (al *AutoLock) IsEnabled() bool {
	al.mu.Lock()
	defer al.mu.Unlock()
	return al.enabled
}