//go:build darwin && !ios

package giode

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
)

// readClipboardImage reads the general pasteboard's image as PNG.
func (a *App) readClipboardImage(ctx context.Context) (image.Image, error) {
	return decodeClipboardImage(clipboardPNG())
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
