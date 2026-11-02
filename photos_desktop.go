//go:build !android && !ios && !js

package giode

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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

// uniquePath adds " (2)", " (3)" ... before the extension until path is
// free.
func uniquePath(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for i := 2; ; i++ {
		p := fmt.Sprintf("%s (%d)%s", base, i, ext)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			return p
		}
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(dst)
		return err
	}
	return out.Close()
}

func sanitizeFileName(s string) string {
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune(`/\:*?"<>|`, r) || r < ' ' {
			return '-'
		}
		return r
	}, s)
}
