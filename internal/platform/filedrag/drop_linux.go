//go:build linux && !android

package filedrag

import "gioui.org/app"

// File drops on Linux come through giode's Gio patches (XDND on X11,
// wl_data_device on Wayland): Gio's own event loop reads the
// protocols' messages.

var installed bool

// Install sets drops up before the window opens.
func Install() { installed = installLinux() }

// ViewReady sets drops up on the window once it exists, and explains
// why they won't arrive, or returns "".
func ViewReady(view app.ViewEvent) string {
	if !installed {
		return "need giode's Gio patches on Linux: build and run the app with an up to date giode CLI (giode upgrade)"
	}
	return ""
}
