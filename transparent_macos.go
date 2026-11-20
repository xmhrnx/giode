//go:build darwin && !ios

package giode

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#include <stdint.h>

void giodeMakeTransparent(uintptr_t view);
*/
import "C"

func (a *App) transparentView() {
	if v := a.NativeHandle(); v != 0 {
		C.giodeMakeTransparent(C.uintptr_t(v))
	}
}
