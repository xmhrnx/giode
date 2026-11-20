//go:build !android && !ios

package picker

import (
	"context"
	"io"

	"github.com/xmhrnx/giode/internal/platform"
)

// Desktops and the web pick photos and media with the file picker.

func PickPhoto(ctx context.Context, h platform.Host, maxSide int, camera bool) (Photo, error) {
	return Photo{}, ErrUseFallback
}

func PickMedia(ctx context.Context, h platform.Host, limit int, photos, videos bool) ([]io.ReadCloser, error) {
	return nil, ErrUseFallback
}
