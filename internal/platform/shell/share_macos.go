//go:build darwin && !ios

package shell

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#include <stdint.h>
#include <stdlib.h>

void giodeSharePicker(uintptr_t view, const char *text, const char *paths);
*/
import "C"

import (
	"strings"
	"unsafe"

	"github.com/xmhrnx/giode/internal/platform"
)

// shareNative shows the system's share picker (AirDrop, Mail, Messages,
// Notes...) under the window's top.
func shareNative(h platform.Host, title, text string, paths []string) error {
	v := h.NativeHandle()
	if v == 0 {
		return platform.ErrNotSupported
	}
	ct, cp := C.CString(text), C.CString(strings.Join(paths, "\n"))
	defer C.free(unsafe.Pointer(ct))
	defer C.free(unsafe.Pointer(cp))
	C.giodeSharePicker(C.uintptr_t(v), ct, cp)
	return nil
}
