//go:build js

package giode

import "context"

func (t ShareText) share(ctx context.Context, a *App) error { return ErrNotSupported }

func (f ShareFile) share(ctx context.Context, a *App) error { return ErrNotSupported }

func (a *App) watchLifecycle() {}

const lifecycleFromWindowState = false

func (f ShareFiles) share(ctx context.Context, a *App) error { return ErrNotSupported }

func (a *App) pickPhotos(ctx context.Context, opts PhotoOptions, limit int) ([]PickedFile, error) {
	return nil, ErrNotSupported
}

func (a *App) saveToPhotosIn(ctx context.Context, path, mime, album string) error {
	return ErrNotSupported
}

func (a *App) saveToDownloads(ctx context.Context, path, mime, folder string) (string, error) {
	return "", ErrNotSupported
}

func (a *App) openDownloads(ctx context.Context, folder string) error { return ErrNotSupported }

func (a *App) openFolder(ctx context.Context, path string) error { return ErrNotSupported }
