//go:build !android && !ios && !js

package giode

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func (a *App) saveToDownloads(ctx context.Context, path, mime, folder string) (string, error) {
	dir, err := downloadsDir()
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
	dir, err := downloadsDir()
	if err != nil {
		return err
	}
	if folder != "" {
		if sub := filepath.Join(dir, folder); isDir(sub) {
			dir = sub
		}
	}
	return a.openFolder(ctx, dir)
}

func (a *App) openFolder(ctx context.Context, path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer.exe", abs)
	case "darwin":
		return workspaceOpen(abs, true)
	default:
		cmd = exec.Command("xdg-open", abs)
	}
	return cmd.Start()
}

func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}
