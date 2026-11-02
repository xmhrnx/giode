package ui

import (
	"gioui.org/io/system"
	"gioui.org/layout"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/i18n"
	"github.com/xmhrnx/giode/state"
)

// Frame lays out one frame of view into gtx the way App does, for a
// custom host (a GLFW window, an embedded view): it starts the frame,
// records the window size and metric, lays the view out, draws the
// portals (menus, tooltips, toasts), delivers the Back key and then the
// keys nothing took to OnKey handlers, runs the
// after-layout callbacks and ends the frame, sweeping keyed state and
// memos no longer used. gtx's constraints are the window's size.
//
// It reports whether another frame is needed right away (a layout
// changed after the fact); the host should then redraw. Animations ask
// for frames through gtx as usual (op.InvalidateCmd, the router's
// wakeup time), and State changes and toasts shown from other
// goroutines through the function given to SetInvalidator.
//
// The host fills the background, hands gtx.Ops to its GPU renderer and
// runs its input router, as in docs/low-level.md.
func Frame(gtx layout.Context, view Element) (needsAnotherFrame bool) {
	elements.BeginFrame()
	defer elements.EndFrame()
	elements.SetWindowSize(gtx.Constraints.Max)
	elements.SetMetric(gtx.Metric)
	if gtx.Locale == (system.Locale{}) {
		gtx.Locale = system.Locale{Language: i18n.Locale(), Direction: i18n.Direction()}
	}
	if view != nil {
		view.Layout(gtx)
	}
	elements.DrainPortals(gtx)
	elements.DispatchBack(gtx)
	// Last, so app shortcuts get only the keys nothing else took.
	elements.DispatchKeys(gtx)
	return elements.RunAfterLayout()
}

// SetInvalidator registers fn, which wakes the host's loop for a new
// frame. It's called from any goroutine: when a State changes, a toast
// is shown from a background goroutine, or anything else outside a
// frame needs a redraw. App sets its own; a custom host sets one that
// wakes its event loop (glfw.PostEmptyEvent, a channel).
func SetInvalidator(fn func()) {
	state.SetInvalidator(fn)
}
