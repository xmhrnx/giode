//go:build js

package giode

import "context"

func (a *App) pickPhotos(ctx context.Context, opts PhotoOptions, limit int) ([]PickedFile, error) {
	return nil, ErrNotSupported
}

func (a *App) saveToPhotosIn(ctx context.Context, path, mime, album string) error {
	return ErrNotSupported
}
