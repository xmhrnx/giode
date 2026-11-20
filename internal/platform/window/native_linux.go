//go:build linux && !android

package window

import (
	"sync/atomic"

	"gioui.org/io/system"

	"github.com/xmhrnx/giode/internal/platform"
)

// Hide unmaps an X11 window. Wayland can't hide a window without
// giode's Gio patches, and a window minimized there can't be brought
// back by the app, so the tray couldn't show it again: there it does
// nothing.
func Hide(h platform.Host) bool {
	if hideX11Window(h.NativeView()) {
		return true
	}
	if waylandSetHidden(h, true) {
		return true
	}
	if !waylandWarned.Swap(true) {
		h.Logf(platform.Warning, "giode: hiding Wayland windows needs giode build's Gio patches; the window stays on screen")
	}
	return false
}

var waylandWarned atomic.Bool

// Show shows a window Hide hid, and raises it.
func Show(h platform.Host) {
	showX11Window(h.NativeView())
	waylandSetHidden(h, false)
	// Gio's Perform must run on the UI goroutine; ShowWindow may not.
	h.Post(func() {
		if !waylandActivate(h) {
			h.Window().Perform(system.ActionRaise)
		}
	})
}

// Raise reports false: raising goes through Gio here.
func Raise(h platform.Host) bool { return false }

// Shown has nothing to set up once the window exists.
func Shown(h platform.Host, interceptClose bool) {}

// InterceptClose: closing goes through Init's closeRequest.
func InterceptClose(h platform.Host) {}

// MenuBarApp is macOS's.
func MenuBarApp() {}
