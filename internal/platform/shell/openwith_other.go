//go:build !android && !ios && !js && !windows && !linux

package shell

import (
	"context"
	"os/exec"
	"runtime"

	"github.com/xmhrnx/giode/internal/platform"
	"github.com/xmhrnx/giode/internal/platform/sandbox"
)

// OpenWith opens the file in its default app.
func OpenWith(ctx context.Context, h platform.Host, path, mime string) error {
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
