package giode

import (
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op/clip"

	"github.com/xmhrnx/giode/elements"
)

// layoutDrag registers a whole-window system-move area so pressing
// empty (non-widget) space drags a frameless window, the way an OS
// titlebar would. It must be laid out before the rest of the view:
// Gio resolves overlapping system.ActionInputOp registrations to
// whichever was registered most recently (see gioui.org/io/input's
// TestPointerSystemAction/"uses topmost action op"), so interactive
// widgets - which each carve out their own bounds with
// kit.BlockDrag/BlockDragHere while they lay themselves out, later in
// the frame - win over this at their own position.
//
// This can't be driven reactively from an observed press the way
// dragging used to be attempted (tracking pointer.Grabbed priority):
// the OS resolves whether a press starts a window-move before
// delivering it as a normal click (on Windows, in the WM_NCHITTEST
// handler, which runs ahead of WM_LBUTTONDOWN), so the registration
// has to be present proactively, every frame, regardless of pointer
// state.
func layoutDrag(a *App, gtx layout.Context) {
	if !a.cfg.Frameless || !a.cfg.Draggable {
		return
	}
	cl := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
	system.ActionInputOp(system.ActionMove).Add(gtx.Ops)
	cl.Pop()
}

// windowPointer is a whole-window pass-through pointer listener for
// the global right-click context menu registered with
// App.SetContextMenu, and for telling touch from mouse input (see
// elements.TouchInput).
//
// It's registered after the rest of the view, not before: PassOp
// keeps it from blocking ordinary (non-pass-through) widgets, but
// does not make it immune to being blocked BY them, and almost every
// pixel of a real UI is covered by some such widget - see
// menu.Context's doc comment for the full reasoning, which applies
// here identically.
type windowPointer struct{}

// layout registers the pointer region and processes events queued
// from the registration on a previous frame: recording whether the
// press was a touch, and opening the context menu on a right-click.
// Only presses are observed, so moving the mouse costs no frames.
func (w *windowPointer) layout(a *App, gtx layout.Context) {
	ps := pointer.PassOp{}.Push(gtx.Ops)
	cl := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
	event.Op(gtx.Ops, w)
	cl.Pop()
	ps.Pop()

	for {
		ev, ok := gtx.Source.Event(pointer.Filter{Target: w, Kinds: pointer.Press})
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		if elements.SetTouchInput(e.Source == pointer.Touch) {
			// Touch targets change size: rebuild cached subtrees and
			// lay out again.
			elements.InvalidateCaches()
			if a.window != nil {
				a.window.Invalidate()
			}
		}
		if a.ctxMenu != nil && e.Buttons.Contain(pointer.ButtonSecondary) {
			a.ctxMenu.OpenAt(e.Position)
		}
	}
}
