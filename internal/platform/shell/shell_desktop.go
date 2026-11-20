//go:build !android && !ios && !js

package shell

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/xmhrnx/giode/internal/platform"
	"github.com/xmhrnx/giode/internal/platform/dirs"
	"github.com/xmhrnx/giode/internal/platform/sandbox"
)

// OpenURL opens u with open (NSWorkspace), xdg-open or the URL handler.
func OpenURL(ctx context.Context, h platform.Host, u string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", u)
	case "darwin":
		return sandbox.Open(u, false)
	default:
		cmd = exec.CommandContext(ctx, "xdg-open", u)
	}
	// The handler keeps running (or hands off and exits); either way
	// there's nothing to wait for.
	return cmd.Start()
}

// OpenFolder opens a folder in Explorer, Finder or the file manager.
func OpenFolder(ctx context.Context, h platform.Host, path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer.exe", abs)
	case "darwin":
		return sandbox.Open(abs, true)
	default:
		cmd = exec.Command("xdg-open", abs)
	}
	return cmd.Start()
}

// SaveToDownloads copies the file into Downloads/<folder>.
func SaveToDownloads(ctx context.Context, path, mime, folder string) (string, error) {
	dir, err := dirs.Downloads()
	if err != nil {
		return "", err
	}
	if folder != "" {
		dir = filepath.Join(dir, folder)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	dst := platform.UniquePath(filepath.Join(dir, filepath.Base(path)))
	return dst, platform.CopyFile(path, dst)
}

// OpenDownloads opens Downloads/<folder>, or Downloads.
func OpenDownloads(ctx context.Context, h platform.Host, folder string) error {
	dir, err := dirs.Downloads()
	if err != nil {
		return err
	}
	if folder != "" {
		if sub := filepath.Join(dir, folder); isDir(sub) {
			dir = sub
		}
	}
	return OpenFolder(ctx, h, dir)
}

// On Windows and macOS sharing opens the system's share sheet; on Linux,
// which has none, text goes to the clipboard (the caller's part, on
// ErrNoShareSheet) and a file is shown in the file manager.

// ShareText opens the share sheet, or returns ErrNoShareSheet.
func ShareText(ctx context.Context, h platform.Host, text, subject, title string) error {
	if err := shareNative(h, title, text, nil); err != nil {
		return ErrNoShareSheet
	}
	return nil
}

// ShareFiles opens the share sheet for the first file, else shows it in
// the file manager.
func ShareFiles(ctx context.Context, h platform.Host, files []File, mime, title string) error {
	if len(files) == 0 {
		return nil
	}
	if len(files) > 1 {
		// Several files share their first, as a single file, as they
		// always have on desktops.
		title = ""
	}
	abs, err := filepath.Abs(files[0].Path)
	if err != nil {
		return err
	}
	if _, err := os.Stat(abs); err != nil {
		return err
	}
	if err := shareNative(h, title, "", []string{abs}); err == nil {
		return nil
	}
	return Reveal(abs)
}
