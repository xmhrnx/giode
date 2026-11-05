//go:build android

package giode

import (
	"context"
	"image"

	"gioui.org/app"

	"github.com/xmhrnx/giode/internal/mobile"
)

// readClipboardImage reads the image content URI on the clipboard, as
// Android's screenshot, gallery and browser apps copy one.
func (a *App) readClipboardImage(ctx context.Context) (image.Image, error) {
	var uri string
	err := a.onAndroidMain(ctx, func(view uintptr) error {
		var err error
		uri, err = mobile.ClipboardImageURI(app.JavaVM(), app.AppContext())
		return err
	})
	if err != nil || uri == "" {
		return nil, err
	}
	data, err := mobile.ReadURI(app.JavaVM(), app.AppContext(), uri, maxClipboardImage)
	if err != nil {
		return nil, err
	}
	return decodeClipboardImage(data)
}
