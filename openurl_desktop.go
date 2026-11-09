//go:build !android && !ios && !js

package giode

import (
	"context"
	"os/exec"
	"runtime"
)

func (a *App) openURL(ctx context.Context, u string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", u)
	case "darwin":
		return workspaceOpen(u, false)
	default:
		cmd = exec.CommandContext(ctx, "xdg-open", u)
	}
	// The handler keeps running (or hands off and exits); either way
	// there's nothing to wait for.
	return cmd.Start()
}
