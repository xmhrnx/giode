//go:build darwin && !ios

package window

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#include <stdint.h>

void giodeWindowButtons(uintptr_t view, int noMin, int noMax, int noClose);
*/
import "C"
import "github.com/xmhrnx/giode/internal/platform"

// Buttons removes the title bar buttons asked for.
func Buttons(h platform.Host, noMin, noMax, noClose bool) {
	if v := h.NativeHandle(); v != 0 {
		b := func(on bool) C.int {
			if on {
				return 1
			}
			return 0
		}
		C.giodeWindowButtons(C.uintptr_t(v), b(noMin), b(noMax), b(noClose))
	}
}
