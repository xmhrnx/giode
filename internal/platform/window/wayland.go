//go:build linux && !android && !nowayland

package window

import "gioui.org/app"

// isWayland reports whether the window is a Wayland one.
func isWayland(view app.ViewEvent) bool {
	_, ok := view.(app.WaylandViewEvent)
	return ok
}
