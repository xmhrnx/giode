//go:build linux && !android && !nox11

package giode

/*
#cgo LDFLAGS: -lX11
#include <X11/Xlib.h>
#include <X11/Xatom.h>

// giodeMotifFunctions sets the window's Motif functions hint, which X11
// window managers (KWin, Mutter, Xfwm, Openbox) read for the buttons
// and actions they offer.
static void giodeMotifFunctions(Display *dpy, Window w, long functions) {
	long hints[5] = {1, functions, 0, 0, 0}; // MWM_HINTS_FUNCTIONS
	Atom a = XInternAtom(dpy, "_MOTIF_WM_HINTS", False);
	XChangeProperty(dpy, w, a, a, 32, PropModeReplace, (unsigned char *)hints, 5);
	XFlush(dpy);
}
*/
import "C"

import "gioui.org/app"

func (a *App) windowButtonsView() {
	x, ok := a.NativeView().(app.X11ViewEvent)
	if !ok || x.Display == nil || x.Window == 0 {
		return
	}
	const (
		funcResize   = 1 << 1
		funcMove     = 1 << 2
		funcMinimize = 1 << 3
		funcMaximize = 1 << 4
		funcClose    = 1 << 5
	)
	f := funcResize | funcMove
	if !a.cfg.NoMinimize {
		f |= funcMinimize
	}
	if !a.cfg.NoMaximize {
		f |= funcMaximize
	}
	if !a.cfg.NoClose {
		f |= funcClose
	}
	C.giodeMotifFunctions((*C.Display)(x.Display), C.Window(x.Window), C.long(f))
}
