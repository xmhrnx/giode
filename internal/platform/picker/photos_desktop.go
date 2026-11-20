//go:build !android && !ios && !js

package picker

import (
	"context"
	"os"
	"path/filepath"

	"github.com/xmhrnx/giode/internal/platform"
	"github.com/xmhrnx/giode/internal/platform/dirs"
)

// PickPhotos: desktops pick image files.
func PickPhotos(ctx context.Context, h platform.Host, maxSide, limit int) ([]Photo, error) {
	return nil, ErrUseFallback
}

// SaveToPhotos copies the file into the user's Pictures/<album>.
func SaveToPhotos(ctx context.Context, path, mime, album string) error {
	dir, err := dirs.Pictures()
	if err != nil {
		return err
	}
	if album != "" {
		dir = filepath.Join(dir, platform.SanitizeFileName(album))
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	dst := platform.UniquePath(filepath.Join(dir, filepath.Base(path)))
	return platform.CopyFile(path, dst)
}
