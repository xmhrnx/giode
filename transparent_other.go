//go:build !windows && !(darwin && !ios)

package giode

// Linux sets transparency through giode's Gio patches before the window
// exists (window_linux_gio8.go); phones and the web don't have it.
func (a *App) transparentView() {}
