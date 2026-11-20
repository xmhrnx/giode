package picker

import (
	"context"

	"github.com/xmhrnx/giode/internal/platform"
)

// The web has no photo library.

func PickPhotos(ctx context.Context, h platform.Host, maxSide, limit int) ([]Photo, error) {
	return nil, platform.ErrNotSupported
}

func SaveToPhotos(ctx context.Context, path, mime, album string) error {
	return platform.ErrNotSupported
}
