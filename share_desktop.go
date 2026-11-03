//go:build !android && !ios && !js

package giode

import (
	"context"
	"os"
	"path/filepath"
)

func (t ShareText) share(ctx context.Context, a *App) error {
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
	return revealFile(abs)
}

func (a *App) watchLifecycle() {}

func (a *App) listenShares() {}

// On desktops, minimizing the window counts as going to the
// background; see handleConfig.
const lifecycleFromWindowState = true
