//go:build ios

package giode

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
)

// documentsDir is the app's Documents folder, which the Files app shows
// with UIFileSharingEnabled and LSSupportsOpeningDocumentsInPlace.
func documentsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Documents"), nil
}

func (a *App) saveToDownloads(ctx context.Context, path, mime, folder string) (string, error) {
	dir, err := documentsDir()
	if err != nil {
		return "", err
	}
	if folder != "" {
		dir = filepath.Join(dir, folder)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	dst := uniquePath(filepath.Join(dir, filepath.Base(path)))
	return dst, copyFile(path, dst)
}

func (a *App) openDownloads(ctx context.Context, folder string) error {
	dir, err := documentsDir()
	if err != nil {
		return err
	}
	if folder != "" {
		if sub := filepath.Join(dir, folder); dirExists(sub) {
			dir = sub
		}
	}
	return a.openFolder(ctx, dir)
}

// openFolder opens a folder in the Files app with its shareddocuments
// URL scheme.
func (a *App) openFolder(ctx context.Context, path string) error {
	u := url.URL{Scheme: "shareddocuments", Path: path}
	return a.openURL(ctx, u.String())
}

func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}
