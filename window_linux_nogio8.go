//go:build linux && !android && !giode_gio8

package giode

const linuxWindowPatched = false

func (a *App) linuxWindowInit()                {}
func (a *App) waylandSetHidden(hide bool) bool { return false }
func (a *App) waylandActivate() bool           { return false }
