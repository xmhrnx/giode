//go:build darwin && !ios

package giode

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#include <stdint.h>

void giodeWindowButtons(uintptr_t view, int noMin, int noMax, int noClose);
*/
import "C"

func (a *App) windowButtonsView() {
	if v := a.NativeHandle(); v != 0 {
		b := func(on bool) C.int {
			if on {
				return 1
			}
			return 0
		}
		C.giodeWindowButtons(C.uintptr_t(v), b(a.cfg.NoMinimize), b(a.cfg.NoMaximize), b(a.cfg.NoClose))
	}
}
