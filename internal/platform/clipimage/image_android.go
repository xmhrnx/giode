//go:build android

package clipimage

import (
	"context"
	"image"

	"gioui.org/app"

	"github.com/xmhrnx/giode/internal/mobile"
	"github.com/xmhrnx/giode/internal/platform"
)

// Read reads the image content URI on the clipboard, as
// Android's screenshot, gallery and browser apps copy one.
func Read(ctx context.Context, h platform.Host) (image.Image, error) {
	var uri string
	err := platform.OnAndroidMain(ctx, h, func(view uintptr) error {
		var err error
		uri, err = mobile.ClipboardImageURI(app.JavaVM(), app.AppContext())
		return err
	})
	if err != nil || uri == "" {
		return nil, err
	}
	data, err := mobile.ReadURI(app.JavaVM(), app.AppContext(), uri, maxImage)
	if err != nil {
		return nil, err
	}
	return decode(data)
}
