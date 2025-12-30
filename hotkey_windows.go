//go:build windows

package main

import (
	"sync"
	"syscall"
	"unsafe"
)

var (
	user32           = syscall.NewLazyDLL("user32.dll")
	registerHotKey   = user32.NewProc("RegisterHotKey")
	unregisterHotKey = user32.NewProc("UnregisterHotKey")
	getMessage       = user32.NewProc("GetMessageW")
	postThreadMsg    = user32.NewProc("PostThreadMessageW")
)

const (
	MOD_CONTROL = 0x0002
	MOD_SHIFT   = 0x0004

	WM_HOTKEY = 0x0312
	WM_QUIT   = 0x0012

	VK_F12 = 0x7B
)

type MSG struct {
	HWND    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

var (
	hotkeyCallback func()
	hotkeyMu       sync.Mutex
	hotkeyRunning  bool
	hotkeyThreadID uintptr
)

// IsGlobalHotkeySupported returns true on Windows
func IsGlobalHotkeySupported() bool {
	return true
}

// GetHotkeyUnavailableReason returns empty string on Windows
func GetHotkeyUnavailableReason() string {
	return ""
}

// RegisterPanicHotkey registers Ctrl+Shift+F12 as panic hotkey
func RegisterPanicHotkey(callback func()) error {
	hotkeyMu.Lock()
	if hotkeyRunning {
		hotkeyMu.Unlock()
		return nil
	}
	hotkeyCallback = callback
	hotkeyRunning = true
	hotkeyMu.Unlock()

	go hotkeyLoop()
	return nil
}

func hotkeyLoop() {
	// Register hotkey in this thread
	ret, _, _ := registerHotKey.Call(0, 1, MOD_CONTROL|MOD_SHIFT, VK_F12)
	if ret == 0 {
		hotkeyMu.Lock()
		hotkeyRunning = false
		hotkeyMu.Unlock()
		return
	}

	var msg MSG
	for {
		ret, _, _ := getMessage.Call(
			uintptr(unsafe.Pointer(&msg)),
			0, 0, 0,
		)

		if ret == 0 || ret == ^uintptr(0) {
			break
		}

		if msg.Message == WM_HOTKEY {
			hotkeyMu.Lock()
			cb := hotkeyCallback
			hotkeyMu.Unlock()

			if cb != nil {
				// Call callback safely
				go cb()
			}
		}

		if msg.Message == WM_QUIT {
			break
		}
	}

	unregisterHotKey.Call(0, 1)

	hotkeyMu.Lock()
	hotkeyRunning = false
	hotkeyMu.Unlock()
}

// UnregisterPanicHotkey unregisters the panic hotkey
func UnregisterPanicHotkey() {
	hotkeyMu.Lock()
	hotkeyCallback = nil
	hotkeyMu.Unlock()

	unregisterHotKey.Call(0, 1)
}
