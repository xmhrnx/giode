//go:build !(android && giode_gio6) && !windows && !darwin && !(linux && !android)

package giode

// installA11y has no screen reader bridge to set up here yet.
func installA11y(a *App) {}
