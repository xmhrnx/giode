//go:build !windows

package giode

// serveContextActions does nothing: only Explorer needs a COM server to
// hand a whole selection to one instance.
func (a *App) serveContextActions() {}

func (a *App) stopContextActions() {}
