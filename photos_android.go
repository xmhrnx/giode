package giode

import (
	"context"

	"gioui.org/app"

	"github.com/xmhrnx/giode/internal/mobile"
)

func (a *App) saveToPhotosIn(ctx context.Context, path, mime, album string) error {
	return mobile.SaveToGallery(app.JavaVM(), app.AppContext(), path, mime, album)
}
