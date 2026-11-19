//go:build !windows && !(darwin && !ios) && !(linux && !android && giode_gio5)

package giode

// Dragging out of the window isn't supported here: drags stay in it.
func installExporter(a *App) {}
