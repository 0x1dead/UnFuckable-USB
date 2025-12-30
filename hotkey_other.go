//go:build !windows

package main

// IsGlobalHotkeySupported returns false on non-Windows platforms
func IsGlobalHotkeySupported() bool {
	return false
}

// GetHotkeyUnavailableReason returns reason why hotkey is unavailable
func GetHotkeyUnavailableReason() string {
	return "global_hotkey_unavailable"
}

// RegisterPanicHotkey is a stub for non-Windows platforms
func RegisterPanicHotkey(callback func()) error {
	return nil
}

// UnregisterPanicHotkey is a stub for non-Windows platforms
func UnregisterPanicHotkey() {
}
