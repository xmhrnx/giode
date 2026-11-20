//go:build darwin && !ios

package window

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework AppKit

#include <stdint.h>

void giodeWatchPlacement(uintptr_t view);
int giodePlacement(double *out);
void giodeSetPosition(uintptr_t view, double x, double y);
*/
import "C"

import "gioui.org/app"

// macOS measures windows in points, which are Gio's dp.

func placement(view app.ViewEvent) ([13]float64, error) {
	var p [13]C.double
	if view == nil || C.giodePlacement(&p[0]) == 0 {
		return [13]float64{}, ErrNoWindowYet
	}
	var out [13]float64
	for i, v := range p {
		out[i] = float64(v)
	}
	return out, nil
}

func rectAt(p []float64) Rect {
	return Rect{X: float32(p[0]), Y: float32(p[1]), Width: float32(p[2]), Height: float32(p[3])}
}

func Frame(view app.ViewEvent, scale float32) (Rect, error) {
	p, err := placement(view)
	return rectAt(p[0:4]), err
}

func ScreenOf(view app.ViewEvent, scale float32) (Screen, error) {
	p, err := placement(view)
	return Screen{Bounds: rectAt(p[4:8]), WorkArea: rectAt(p[8:12]), Scale: float32(p[12])}, err
}

func SetPosition(view app.ViewEvent, x, y, scale float32) error {
	ev, ok := view.(app.AppKitViewEvent)
	if !ok || ev.View == 0 {
		return ErrNoWindowYet
	}
	C.giodeSetPosition(C.uintptr_t(ev.View), C.double(x), C.double(y))
	return nil
}

// SetTopMost leaves always on top to Gio.
func SetTopMost(view app.ViewEvent, on bool) bool { return false }

// Placed watches the window's moves.
func Placed(ev app.ViewEvent, topMost bool) {
	if v, ok := ev.(app.AppKitViewEvent); ok && v.View != 0 {
		C.giodeWatchPlacement(C.uintptr_t(v.View))
	}
}

func ClientOrigin(view app.ViewEvent) (x, y int, ok bool) { return 0, 0, false }
