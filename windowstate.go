package giode

import (
	"gioui.org/app"

	"github.com/xmhrnx/giode/elements"
)

// WindowState is the current state of the window, as reported by the
// platform. It changes through App methods (Maximize, Minimize,
// SetFullscreen, ...) as well as through the user (a double-click on
// the title bar, snapping, the taskbar).
type WindowState int

const (
	// WindowNormal is a regular, restored window.
	WindowNormal WindowState = iota
	// WindowMinimized is a window minimized to the taskbar or dock.
	WindowMinimized
	// WindowMaximized fills the available screen area.
	WindowMaximized
	// WindowFullscreen covers the whole screen.
	WindowFullscreen
)

// String returns the state's name.
func (s WindowState) String() string {
	switch s {
	case WindowMinimized:
		return "minimized"
	case WindowMaximized:
		return "maximized"
	case WindowFullscreen:
		return "fullscreen"
	default:
		return "normal"
	}
}

// WindowState returns the window's current state. It is safe to call
// from any goroutine.
func (a *App) WindowState() WindowState {
	return WindowState(a.winState.Load())
}

// Maximized reports whether the window is maximized.
func (a *App) Maximized() bool {
	return a.WindowState() == WindowMaximized
}

// Fullscreen reports whether the window is fullscreen.
func (a *App) Fullscreen() bool {
	return a.WindowState() == WindowFullscreen
}

// Focused reports whether the window has keyboard focus.
func (a *App) Focused() bool {
	return a.winFocused.Load()
}

// ToggleMaximize maximizes a restored window and restores a maximized
// (or fullscreen) one - the action behind a title bar's maximize
// button.
func (a *App) ToggleMaximize() {
	switch a.WindowState() {
	case WindowMaximized:
		a.Unmaximize()
	case WindowFullscreen:
		a.SetFullscreen(false)
	default:
		a.Maximize()
	}
}

// handleConfig records window state and focus changes reported by the
// platform, notifies OnWindowStateChange and makes cached UI (a title
// bar's maximize/restore glyph, focus styling) repaint.
func (a *App) handleConfig(cfg app.Config) {
	var s WindowState
	switch cfg.Mode {
	case app.Minimized:
		s = WindowMinimized
	case app.Maximized:
		s = WindowMaximized
	case app.Fullscreen:
		s = WindowFullscreen
	default:
		s = WindowNormal
	}
	old := WindowState(a.winState.Swap(int32(s)))
	focusChanged := a.winFocused.Swap(cfg.Focused) != cfg.Focused
	if focusChanged && cfg.Focused {
		a.checkSystemTheme()
	}
	if old == s && !focusChanged {
		return
	}
	elements.InvalidateCaches()
	a.invalidate()
	if old != s {
		a.logf(LogLevelDebug, "giode: window %s", s)
		if s == WindowMinimized && a.minimizesToTray() && !a.windowHidden.Load() {
			// Minimized to the tray: off the taskbar or Dock too.
			if a.windowHidden.CompareAndSwap(false, true) && !a.hideNativeWindow() {
				a.windowHidden.Store(false)
			}
		}
		if a.cfg.OnWindowStateChange != nil {
			a.cfg.OnWindowStateChange(s)
		}
		if lifecycleFromWindowState {
			switch {
			case s == WindowMinimized && a.cfg.OnBackground != nil:
				a.cfg.OnBackground()
			case old == WindowMinimized && a.cfg.OnForeground != nil:
				a.cfg.OnForeground()
			}
		}
	}
}
