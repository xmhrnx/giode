//go:build linux && !android && !nox11

package window

/*
#cgo LDFLAGS: -lX11 -ldl
#include <stdlib.h>
#include <dlfcn.h>
#include <X11/Xlib.h>
#include <X11/Xatom.h>

// XRRMonitorInfo, from Xrandr.h: libXrandr is loaded at run time, so
// building needs neither it nor its headers.
typedef struct {
	Atom name;
	Bool primary, automatic;
	int noutput;
	int x, y, width, height;
	int mwidth, mheight;
	unsigned long *outputs;
} giodeMonitor;

typedef giodeMonitor *(*giodeGetMonitors)(Display *, Window, Bool, int *);
typedef void (*giodeFreeMonitors)(giodeMonitor *);

// giode_monitor finds the monitor containing (cx, cy) in root window
// coordinates, or the first, reporting whether XRandR 1.5 answered.
static int giode_monitor(Display *dpy, int cx, int cy, int *x, int *y, int *w, int *h) {
	static void *lib;
	static giodeGetMonitors get;
	static giodeFreeMonitors freeMon;
	if (!lib) {
		lib = dlopen("libXrandr.so.2", RTLD_LAZY);
		if (!lib) return 0;
		get = (giodeGetMonitors)dlsym(lib, "XRRGetMonitors");
		freeMon = (giodeFreeMonitors)dlsym(lib, "XRRFreeMonitors");
	}
	if (!get || !freeMon) return 0;
	int n = 0;
	giodeMonitor *m = get(dpy, DefaultRootWindow(dpy), True, &n);
	if (!m) return 0;
	int found = n > 0 ? 0 : -1;
	for (int i = 0; i < n; i++) {
		if (cx >= m[i].x && cx < m[i].x + m[i].width && cy >= m[i].y && cy < m[i].y + m[i].height) {
			found = i;
			break;
		}
	}
	if (found >= 0) {
		*x = m[found].x; *y = m[found].y; *w = m[found].width; *h = m[found].height;
	}
	freeMon(m);
	return found >= 0;
}

// giode_cardinals reads a CARDINAL[] property into out, returning how
// many it read.
static int giode_cardinals(Display *dpy, Window win, const char *name, long *out, int max) {
	Atom prop = XInternAtom(dpy, name, True);
	if (prop == None) return 0;
	Atom type;
	int format;
	unsigned long n, after;
	unsigned char *data = NULL;
	if (XGetWindowProperty(dpy, win, prop, 0, max, False, XA_CARDINAL, &type, &format, &n, &after, &data) != Success || !data) return 0;
	int got = 0;
	if (format == 32) {
		long *l = (long *)data;
		for (; got < (int)n && got < max; got++) out[got] = l[got];
	}
	XFree(data);
	return got;
}

static void giode_set_above(Display *dpy, Window win, int on) {
	XEvent ev = {0};
	ev.xclient.type = ClientMessage;
	ev.xclient.window = win;
	ev.xclient.message_type = XInternAtom(dpy, "_NET_WM_STATE", False);
	ev.xclient.format = 32;
	ev.xclient.data.l[0] = on ? 1 : 0; // _NET_WM_STATE_ADD, _REMOVE
	ev.xclient.data.l[1] = XInternAtom(dpy, "_NET_WM_STATE_ABOVE", False);
	ev.xclient.data.l[3] = 1; // from an application
	XSendEvent(dpy, DefaultRootWindow(dpy), False, SubstructureRedirectMask | SubstructureNotifyMask, &ev);
	XFlush(dpy);
}
*/
import "C"

import (
	"math"
	"unsafe"

	"gioui.org/app"
)

func x11Window(ev app.ViewEvent) (*C.Display, C.Window, bool) {
	x, ok := ev.(app.X11ViewEvent)
	if !ok || x.Display == nil || x.Window == 0 {
		return nil, 0, false
	}
	return (*C.Display)(x.Display), C.Window(x.Window), true
}

// x11Outer returns the window's frame in root pixels: the client area
// plus the window manager's frame (_NET_FRAME_EXTENTS).
func x11Outer(dpy *C.Display, win C.Window) (x, y, w, h int) {
	var cx, cy C.int
	var child C.Window
	C.XTranslateCoordinates(dpy, win, C.XDefaultRootWindow(dpy), 0, 0, &cx, &cy, &child)
	var attrs C.XWindowAttributes
	C.XGetWindowAttributes(dpy, win, &attrs)
	var ext [4]C.long // left, right, top, bottom
	name := C.CString("_NET_FRAME_EXTENTS")
	defer C.free(unsafe.Pointer(name))
	if C.giode_cardinals(dpy, win, name, &ext[0], 4) != 4 {
		ext = [4]C.long{}
	}
	return int(cx) - int(ext[0]), int(cy) - int(ext[2]), int(attrs.width) + int(ext[0]+ext[1]), int(attrs.height) + int(ext[2]+ext[3])
}

func x11Frame(ev app.ViewEvent, s float32) (Rect, error) {
	dpy, win, ok := x11Window(ev)
	if !ok {
		return Rect{}, ErrNoWindowYet
	}
	x, y, w, h := x11Outer(dpy, win)
	return pxRect(x, y, w, h, s), nil
}

func x11SetPosition(ev app.ViewEvent, x, y, s float32) error {
	dpy, win, ok := x11Window(ev)
	if !ok {
		return ErrNoWindowYet
	}
	// With the default (north west) gravity, window managers place the
	// frame's corner at the position.
	C.XMoveWindow(dpy, win, C.int(math.Round(float64(x*s))), C.int(math.Round(float64(y*s))))
	C.XFlush(dpy)
	return nil
}

func x11Screen(ev app.ViewEvent, s float32) (Screen, error) {
	dpy, win, ok := x11Window(ev)
	if !ok {
		return Screen{}, ErrNoWindowYet
	}
	ox, oy, ow, oh := x11Outer(dpy, win)
	root := C.XDefaultRootWindow(dpy)
	var mx, my, mw, mh C.int
	if C.giode_monitor(dpy, C.int(ox+ow/2), C.int(oy+oh/2), &mx, &my, &mw, &mh) == 0 {
		scr := C.XDefaultScreen(dpy)
		mx, my, mw, mh = 0, 0, C.XDisplayWidth(dpy, scr), C.XDisplayHeight(dpy, scr)
	}
	bounds := [4]int{int(mx), int(my), int(mw), int(mh)}
	work := bounds
	// _NET_WORKAREA spans every monitor, less the panels at the
	// desktop's edges; within this monitor, that's its work area.
	var desk [1]C.long
	cur := C.CString("_NET_CURRENT_DESKTOP")
	wa := C.CString("_NET_WORKAREA")
	defer C.free(unsafe.Pointer(cur))
	defer C.free(unsafe.Pointer(wa))
	d := 0
	if C.giode_cardinals(dpy, root, cur, &desk[0], 1) == 1 {
		d = int(desk[0])
	}
	areas := make([]C.long, 4*(d+1))
	if C.giode_cardinals(dpy, root, wa, &areas[0], C.int(len(areas))) == C.int(len(areas)) {
		a := areas[4*d:]
		x0, y0 := max(bounds[0], int(a[0])), max(bounds[1], int(a[1]))
		x1, y1 := min(bounds[0]+bounds[2], int(a[0]+a[2])), min(bounds[1]+bounds[3], int(a[1]+a[3]))
		if x1 > x0 && y1 > y0 {
			work = [4]int{x0, y0, x1 - x0, y1 - y0}
		}
	}
	return Screen{Bounds: pxRect(bounds[0], bounds[1], bounds[2], bounds[3], s), WorkArea: pxRect(work[0], work[1], work[2], work[3], s), Scale: s}, nil
}

func x11SetAbove(ev app.ViewEvent, on bool) {
	if dpy, win, ok := x11Window(ev); ok {
		v := C.int(0)
		if on {
			v = 1
		}
		C.giode_set_above(dpy, win, v)
	}
}

func pxRect(x, y, w, h int, s float32) Rect {
	return Rect{X: float32(x) / s, Y: float32(y) / s, Width: float32(w) / s, Height: float32(h) / s}
}

// x11ClientOrigin returns the window's client area origin in root
// pixels, for screen readers asking for screen coordinates.
func x11ClientOrigin(ev app.ViewEvent) (x, y int, ok bool) {
	dpy, win, ok := x11Window(ev)
	if !ok {
		return 0, 0, false
	}
	var cx, cy C.int
	var child C.Window
	C.XTranslateCoordinates(dpy, win, C.XDefaultRootWindow(dpy), 0, 0, &cx, &cy, &child)
	return int(cx), int(cy), true
}
