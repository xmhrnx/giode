//go:build ios

package giode

/*
#include <stdlib.h>

void *giodeClipboardImagePNG(int *n);
*/
import "C"

import (
	"context"
	"image"
)

// readClipboardImage reads the general pasteboard's image as PNG. iOS
// 16 and later show a banner (or ask) when an app reads what another
// app copied.
func (a *App) readClipboardImage(ctx context.Context) (image.Image, error) {
	var n C.int
	p := C.giodeClipboardImagePNG(&n)
	if p == nil {
		return nil, nil
	}
	defer C.free(p)
	return decodeClipboardImage(C.GoBytes(p, n))
}
