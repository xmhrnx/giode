//go:build !android && !ios && !js

package giode

import (
	"context"
	"os"
	"path/filepath"
)

// On Windows and macOS sharing opens the system's share sheet; on Linux,
// which has none, text goes to the clipboard and a file is shown in the
// file manager.

func (t ShareText) share(ctx context.Context, a *App) error {
	if err := a.shareNative(t.Title, t.Text, nil); err == nil {
		return nil
	}
	a.SetClipboard(t.Text)
	return nil
}

func (f ShareFile) share(ctx context.Context, a *App) error {
	abs, err := filepath.Abs(f.Path)
	if err != nil {
		return err
	}
	if _, err := os.Stat(abs); err != nil {
		return err
	}
	if err := a.shareNative(f.Title, "", []string{abs}); err == nil {
		return nil
	}
	return revealFile(abs)
}

func (a *App) watchLifecycle() {}

func (a *App) listenShares() {}

// On desktops, minimizing the window counts as going to the
// background; see handleConfig.
const lifecycleFromWindowState = true
