package giode

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// maxClipboardImage bounds the encoded image ReadClipboardImage reads.
const maxClipboardImage = 64 << 20

// ReadClipboardImage returns the image on the system clipboard - a
// screenshot, or an image copied from a browser, a document or an image
// editor - or nil when the clipboard holds no image. It returns
// ErrNotSupported where giode can't read images from the clipboard.
//
// It reads the clipboard directly and may wait for the app that owns
// it, so call it from a goroutine, like OpenFile:
//
//	go func() {
//		img, err := app.ReadClipboardImage(ctx)
//		if err != nil || img == nil {
//			return
//		}
//		...
//	}()
//
// Where it comes from: Windows' clipboard (PNG, else a device-
// independent bitmap); macOS's and iOS's pasteboards; Android's
// clipboard (an image content URI, as Android's screenshot and photo
// apps copy); the browser's async clipboard (the page must be focused,
// and the browser may ask the user); on Linux, wl-paste (Wayland) or
// xclip (X11), which must be installed.
func (a *App) ReadClipboardImage(ctx context.Context) (image.Image, error) {
	return a.readClipboardImage(ctx)
}

// decodeClipboardImage decodes an image read from the clipboard in any
// format Go can: PNG, JPEG, GIF, BMP, TIFF, WebP.
func decodeClipboardImage(data []byte) (image.Image, error) {
	if len(data) == 0 {
		return nil, nil
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("giode: decoding the clipboard image: %w", err)
	}
	return img, nil
}
