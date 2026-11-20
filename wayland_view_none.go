//go:build linux && !android && nowayland

package giode

func (a *App) isWayland() bool { return false }
