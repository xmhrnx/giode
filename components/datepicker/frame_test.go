package datepicker

import (
	"gioui.org/layout"

	"github.com/xmhrnx/giode/elements"
)

// frame runs a frame's steps as ui.Frame does (ui can't be imported
// here: it re-exports this package).
func frame(gtx layout.Context, view elements.Element) {
	elements.BeginFrame()
	defer elements.EndFrame()
	elements.SetWindowSize(gtx.Constraints.Max)
	elements.SetMetric(gtx.Metric)
	view.Layout(gtx)
	elements.DrainContextMenus(gtx)
	elements.DrainPortals(gtx)
	elements.DispatchBack(gtx)
	elements.DispatchKeys(gtx)
	elements.RunAfterLayout()
}
