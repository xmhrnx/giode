//go:build linux && !android

package giode

import (
	"sync/atomic"

	"gioui.org/app"
	"gioui.org/io/system"

	"fyne.io/systray"
)

// setTrayIcon sets the tray icon from PNG bytes.
func setTrayIcon(icon []byte, _ bool) {
	systray.SetIcon(icon)
}

// hideNativeWindow unmaps an X11 window. Wayland can't hide a window,
// and a window minimized there can't be brought back by the app, so
// the tray couldn't show it again: there it does nothing.
func (a *App) hideNativeWindow() bool {
	if hideX11Window(a.NativeView()) {
		return true
	}
	if !trayWaylandWarned.Swap(true) {
		a.logf(LogLevelWarning, "giode: Wayland can't hide windows; the window stays on screen")
	}
	return false
}

var trayWaylandWarned atomic.Bool

func (a *App) showNativeWindow() {
	showX11Window(a.NativeView())
	a.window.Perform(system.ActionRaise)
}

// trayCloseWarned limits the CloseToTray warning to once.
var trayCloseWarned atomic.Bool

func (a *App) warnTrayLimits() {
	if a.closeToTray.Load() && trayCloseWarned.CompareAndSwap(false, true) {
		a.logf(LogLevelWarning, "giode: CloseToTray isn't supported on Linux; closing the window quits")
	}
}

func (a *App) platformInit()          {}
func (a *App) trayView(app.ViewEvent) {}
func (a *App) applyTrayOptions()      { a.warnTrayLimits() }
