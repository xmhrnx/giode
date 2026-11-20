//go:build linux && !android && giode_gio8

package window

import (
	"os"

	"gioui.org/app"

	"github.com/xmhrnx/giode/internal/platform"
)

// Built against giode's Gio patches (giode build): the window manager's
// close goes through OnCloseRequest and CloseToTray, Wayland windows hide,
// raise and can be transparent.

// CloseNeedsPatch reports whether CloseToTray can't intercept closing
// here: Linux without giode's Gio patches.
const CloseNeedsPatch = false

// Init sets the window up before it opens: transparency, and closing
// through closeRequest, which reports whether the window may close.
func Init(h platform.Host, transparent bool, closeRequest func() bool) {
	app.GiodeSetTransparent(transparent)
	app.GiodeSetCloseRequest(closeRequest)
}

// waylandSetHidden hides or shows a Wayland window, on the UI goroutine.
func waylandSetHidden(h platform.Host, hide bool) bool {
	if !isWayland(h.NativeView()) {
		return false
	}
	h.Post(func() { h.Window().GiodeSetHidden(hide) })
	return true
}

// waylandActivate raises and focuses a Wayland window with
// xdg-activation, the first time with the token the app was launched
// with; on the UI goroutine.
func waylandActivate(h platform.Host) bool {
	if !isWayland(h.NativeView()) {
		return false
	}
	token := os.Getenv("XDG_ACTIVATION_TOKEN")
	os.Unsetenv("XDG_ACTIVATION_TOKEN")
	return h.Window().GiodeActivate(token)
}
