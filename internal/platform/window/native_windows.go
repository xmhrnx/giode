package window

import "github.com/xmhrnx/giode/internal/platform"

var (
	procShowWindowAsync     = user32.NewProc("ShowWindowAsync")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procIsIconic            = user32.NewProc("IsIconic")
	procBringWindowToTop    = user32.NewProc("BringWindowToTop")
)

const (
	swHide    = 0
	swShow    = 5
	swRestore = 9
)

// CloseNeedsPatch is Linux's.
const CloseNeedsPatch = false

// Init: closing goes through the window procedure the app subclasses.
func Init(h platform.Host, transparent bool, closeRequest func() bool) {}

// MenuBarApp is macOS's.
func MenuBarApp() {}

// The Async variant posts to the window's thread: a synchronous call
// from the UI goroutine would wait on Gio's window thread, which may
// be waiting on the UI goroutine.

// Hide hides the window, off the taskbar.
func Hide(h platform.Host) bool {
	hwnd := h.NativeHandle()
	if hwnd == 0 {
		return false
	}
	HideHWND(hwnd)
	return true
}

// HideHWND hides a window, from its own window procedure too.
func HideHWND(hwnd uintptr) {
	procShowWindowAsync.Call(hwnd, swHide)
}

// Raise restores the window if it's minimized and brings it to the
// foreground, through its HWND, so it works from any goroutine and
// while Gio draws no frames (a minimized window gets none).
func Raise(h platform.Host) bool {
	hwnd := h.NativeHandle()
	if hwnd == 0 {
		return false
	}
	if iconic, _, _ := procIsIconic.Call(hwnd); iconic != 0 {
		procShowWindowAsync.Call(hwnd, swRestore)
	}
	go func() {
		procSetForegroundWindow.Call(hwnd)
		procBringWindowToTop.Call(hwnd)
	}()
	return true
}

// Show shows a hidden window and brings it forward.
func Show(h platform.Host) {
	hwnd := h.NativeHandle()
	if hwnd == 0 {
		return
	}
	cmd := uintptr(swShow)
	if iconic, _, _ := procIsIconic.Call(hwnd); iconic != 0 {
		cmd = swRestore
	}
	procShowWindowAsync.Call(hwnd, cmd)
	go procSetForegroundWindow.Call(hwnd)
}

// Shown: the window procedure handles closing.
func Shown(h platform.Host, interceptClose bool) {}

// InterceptClose: the window procedure handles closing.
func InterceptClose(h platform.Host) {}
