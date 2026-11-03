//go:build !android && !ios && !js

package giode

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

func (f ShareFiles) share(ctx context.Context, a *App) error {
	if len(f.Paths) == 0 {
		return nil
	}
	return ShareFile{Path: f.Paths[0]}.share(ctx, a)
}

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
	dir, err := picturesDir()
	if err != nil {
		return err
	}
	if album != "" {
		dir = filepath.Join(dir, sanitizeFileName(album))
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	dst := uniquePath(filepath.Join(dir, filepath.Base(path)))
	return copyFile(path, dst)
}
