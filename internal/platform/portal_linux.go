//go:build linux && !android

package platform

import (
	"fmt"

	"gioui.org/app"
)

// PortalParent is the window the XDG desktop portal's dialogs belong to:
// "x11:<id>" for an X11 window, "" otherwise (or without a host).
func PortalParent(h Host) string {
	if h == nil {
		return ""
	}
	if v, ok := h.NativeView().(app.X11ViewEvent); ok && v.Window != 0 {
		return fmt.Sprintf("x11:%x", v.Window)
	}
	return ""
}
