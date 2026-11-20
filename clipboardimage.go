package giode

import (
	"context"
	"image"

	"github.com/xmhrnx/giode/internal/platform/clipimage"
)

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
	return clipimage.Read(ctx, a.host())
}
