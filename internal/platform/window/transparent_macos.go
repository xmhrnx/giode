//go:build darwin && !ios

package window

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#include <stdint.h>

void giodeMakeTransparent(uintptr_t view);
*/
import "C"
import "github.com/xmhrnx/giode/internal/platform"

// MakeTransparent lets the window show through where the UI is
// transparent.
func MakeTransparent(h platform.Host) {
	if v := h.NativeHandle(); v != 0 {
		C.giodeMakeTransparent(C.uintptr_t(v))
	}
}
