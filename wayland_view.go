//go:build linux && !android && !nowayland

package giode

import "gioui.org/app"

// isWayland reports whether the window is a Wayland one.
func (a *App) isWayland() bool {
	_, ok := a.NativeView().(app.WaylandViewEvent)
	return ok
}
