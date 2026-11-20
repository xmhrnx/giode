//go:build linux && !android && giode_gio8

package giode

import (
	"os"

	"gioui.org/app"
)

// Built against giode's Gio patches (giode build): the window manager's
// close goes through OnCloseRequest and CloseToTray, Wayland windows hide,
// raise and can be transparent.

const linuxWindowPatched = true

func (a *App) linuxWindowInit() {
	app.GiodeSetTransparent(a.cfg.Transparent)
	app.GiodeSetCloseRequest(func() bool {
		if a.vetoClose() {
			return false
		}
		if a.closesToTray() {
			a.HideWindow()
			return false
		}
		return true
	})
}

// waylandSetHidden hides or shows a Wayland window, on the UI goroutine.
func (a *App) waylandSetHidden(hide bool) bool {
	if !a.isWayland() {
		return false
	}
	a.Post(func() { a.window.GiodeSetHidden(hide) })
	return true
}

// waylandActivate raises and focuses a Wayland window with
// xdg-activation, the first time with the token the app was launched
// with; on the UI goroutine.
func (a *App) waylandActivate() bool {
	if !a.isWayland() {
		return false
	}
	token := os.Getenv("XDG_ACTIVATION_TOKEN")
	os.Unsetenv("XDG_ACTIVATION_TOKEN")
	return a.window.GiodeActivate(token)
}
