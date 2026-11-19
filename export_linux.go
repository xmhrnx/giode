//go:build giode_gio5 && linux && !android

package giode

import (
	"gioui.org/app"

	"github.com/xmhrnx/giode/elements/dnd"
)

// Dragging out of the window on Linux (Draggable.Export) comes through
// giode's Gio patches: an XDND source on X11, wl_data_device.start_drag
// on Wayland, both run by Gio's event loop.
func installExporter(a *App) {
	dnd.SetExporter(func(data dnd.ExportData, done func(bool)) bool {
		return app.GiodeStartDrag(data.Files, data.Text, func(dropped bool) {
			a.Post(func() { done(dropped) })
		})
	})
}
