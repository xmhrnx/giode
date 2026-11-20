//go:build !android && !ios && !js && !windows && !linux

package giode

import (
	"context"
	"os/exec"
	"runtime"

	"github.com/xmhrnx/giode/internal/platform/sandbox"
)

func (a *App) openWith(ctx context.Context, path, mime string) error {
	if runtime.GOOS == "darwin" {
		return sandbox.Open(path, true)
	}
	// The handler keeps running (or hands off and exits); either way
	// there's nothing to wait for.
	cmd := exec.Command("xdg-open", path)
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}
