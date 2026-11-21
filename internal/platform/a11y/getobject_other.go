//go:build !windows

package a11y

// GetObject and WindowGone are UI Automation's, on Windows.
func GetObject(a Host, hwnd, wparam, lparam uintptr) (uintptr, bool) { return 0, false }
func WindowGone(hwnd uintptr)                                        {}
