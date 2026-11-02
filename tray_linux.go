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

func (a *App) hideNativeWindow() {
	if !hideX11Window(a.NativeView()) {
		// Wayland has no way to hide a window: minimize it.
		a.window.Perform(system.ActionMinimize)
	}
}

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
