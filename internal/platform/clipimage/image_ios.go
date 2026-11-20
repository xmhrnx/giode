//go:build ios

package clipimage

/*
#cgo CFLAGS: -Werror -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework UIKit

#include <stdlib.h>

void *giodeClipboardImagePNG(int *n);
*/
import "C"

import (
	"context"
	"image"

	"github.com/xmhrnx/giode/internal/platform"
)

// Read reads the general pasteboard's image as PNG. iOS
// 16 and later show a banner (or ask) when an app reads what another
// app copied.
func Read(ctx context.Context, h platform.Host) (image.Image, error) {
	var n C.int
	p := C.giodeClipboardImagePNG(&n)
	if p == nil {
		return nil, nil
	}
	defer C.free(p)
	return decode(C.GoBytes(p, n))
}
