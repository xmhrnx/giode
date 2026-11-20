//go:build darwin && !ios

package clipimage

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework AppKit

#include <stdlib.h>

void *giodeClipboardImagePNG(int *n);
*/
import "C"

import (
	"context"
	"image"

	"github.com/xmhrnx/giode/internal/platform"
)

// Read reads the general pasteboard's image as PNG.
func Read(ctx context.Context, h platform.Host) (image.Image, error) {
	return decode(clipboardPNG())
}

func clipboardPNG() []byte {
	var n C.int
	p := C.giodeClipboardImagePNG(&n)
	if p == nil {
		return nil
	}
	defer C.free(p)
	return C.GoBytes(p, n)
}
