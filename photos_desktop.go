//go:build !android && !ios && !js

package giode

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xmhrnx/giode/internal/platform"
	"github.com/xmhrnx/giode/internal/platform/dirs"
)

func (a *App) pickPhotos(ctx context.Context, opts PhotoOptions, limit int) ([]PickedFile, error) {
	files, err := a.PickFiles(ctx, PickOptions{Multiple: true, Types: []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}})
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(files) > limit {
		for _, f := range files[limit:] {
			f.Close()
		}
		files = files[:limit]
	}
	var out []PickedFile
	for _, f := range files {
		data, err := toJPEG(f, opts.MaxSide)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f.Name, err)
		}
		out = append(out, jpegFile(f.Name, data))
	}
	if len(out) == 0 {
		return nil, ErrCanceled
	}
	return out, nil
}

func (a *App) saveToPhotosIn(ctx context.Context, path, mime, album string) error {
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
