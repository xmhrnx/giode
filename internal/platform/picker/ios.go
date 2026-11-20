//go:build ios

package picker

/*
#include <CoreFoundation/CoreFoundation.h>
*/
import "C"

import "github.com/xmhrnx/giode/internal/platform"

// viewController is the UIViewController behind the window.
func viewController(h platform.Host) (C.CFTypeRef, error) {
	v := h.NativeHandle()
	if v == 0 {
		return 0, platform.ErrNotSupported
	}
	return C.CFTypeRef(v), nil
}
