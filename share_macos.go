//go:build darwin && !ios

package giode

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
)

// shareNative shows the system's share picker (AirDrop, Mail, Messages,
// Notes...) under the window's top.
func (a *App) shareNative(title, text string, paths []string) error {
	v := a.NativeHandle()
	if v == 0 {
		return ErrNotSupported
	}
	ct, cp := C.CString(text), C.CString(strings.Join(paths, "\n"))
	defer C.free(unsafe.Pointer(ct))
	defer C.free(unsafe.Pointer(cp))
	C.giodeSharePicker(C.uintptr_t(v), ct, cp)
	return nil
}
