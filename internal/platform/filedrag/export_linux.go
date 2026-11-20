//go:build giode_gio5 && linux && !android

package filedrag

import (
	"gioui.org/app"

	"github.com/xmhrnx/giode/elements/dnd"
	"github.com/xmhrnx/giode/internal/platform"
)

// Dragging out of the window on Linux (Draggable.Export) comes through
// giode's Gio patches: an XDND source on X11, wl_data_device.start_drag
// on Wayland, both run by Gio's event loop.
// InstallExporter lets Draggable.Export drag out of the window; done
// runs through post, on the UI goroutine.
func InstallExporter(h platform.Host) {
	dnd.SetExporter(func(data dnd.ExportData, done func(bool)) bool {
		return app.GiodeStartDrag(data.Files, data.Text, func(dropped bool) {
			h.Post(func() { done(dropped) })
		})
	})
}
