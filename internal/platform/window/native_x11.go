//go:build linux && !android && !nox11

package window

/*
#cgo LDFLAGS: -lX11
#include <X11/Xlib.h>
*/
import "C"

import (
	"gioui.org/app"
)

// hideX11Window unmaps an X11 window, which takes it off the screen and
// the taskbar. It reports false for a window that isn't X11.
func hideX11Window(ev app.ViewEvent) bool {
	x, ok := ev.(app.X11ViewEvent)
	if !ok || x.Display == nil || x.Window == 0 {
		return false
	}
	dpy := (*C.Display)(x.Display)
	C.XUnmapWindow(dpy, C.Window(x.Window))
	C.XFlush(dpy)
	return true
}

// showX11Window maps an X11 window unmapped by hideX11Window.
func showX11Window(ev app.ViewEvent) {
	x, ok := ev.(app.X11ViewEvent)
	if !ok || x.Display == nil || x.Window == 0 {
		return
	}
	dpy := (*C.Display)(x.Display)
	C.XMapRaised(dpy, C.Window(x.Window))
	C.XFlush(dpy)
}
