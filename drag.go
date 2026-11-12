package giode

import (
	"image"

	"gioui.org/io/event"
	"gioui.org/io/key"
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
// pixel of a real UI is covered by some such widget, and Gio's
// hit-test stops at the first one it meets: registered last, the
// listener sits in front of them and sees every press, while still
// leaving them their own.
type windowPointer struct {
	// The touch press being watched for a tap: where, and whether the
	// focused text field (the keyboard's owner then) keeps the keyboard
	// for it.
	tapAt    image.Point
	tapOwner elements.KeyboardOwner
}

// layout registers the pointer region and processes events queued
// from the registration on a previous frame: recording whether the
// press was a touch, and opening the context menu on a right-click.
// Presses and releases are observed, not moves, so moving the mouse
// costs no frames.
//
// Releases are taken (and ignored) so that every release has a
// handler. Gio keeps an event only when some handler receives it; one
// nobody receives is folded into the router's state. When a command
// runs during a frame (a caret blink's InvalidateCmd), Gio rewinds that
// state and replays the kept events - the press over empty space
// without its release - and so believes the button is still held: the
// next press then reaches only this listener, not the widget under it.
func (w *windowPointer) layout(a *App, gtx layout.Context) {
	ps := pointer.PassOp{}.Push(gtx.Ops)
	cl := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
	event.Op(gtx.Ops, w)
	cl.Pop()
	ps.Pop()

	owner := elements.TakeKeyboardOwner()
	for {
		ev, ok := gtx.Source.Event(pointer.Filter{Target: w, Kinds: pointer.Press | pointer.Release})
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		if e.Kind == pointer.Release {
			w.tapped(gtx, e, owner)
			continue
		}
		if e.Kind != pointer.Press {
			continue
		}
		w.tapOwner = nil
		if e.Source == pointer.Touch && owner != nil && !owner.KeepsKeyboard(e.Position.Round()) {
			w.tapAt, w.tapOwner = e.Position.Round(), owner
		}
		if elements.SetTouchInput(e.Source == pointer.Touch) {
			// Touch targets change size: rebuild cached subtrees and
			// lay out again.
			elements.InvalidateCaches()
			if a.window != nil {
				a.window.Invalidate()
			}
		}
		// A press an element opened its own context menu for isn't the
		// window's.
		if a.ctxMenu != nil && e.Buttons.Contain(pointer.ButtonSecondary) && !elements.ContextClaimed(e) {
			a.ctxMenu.OpenAt(e.Position)
		}
	}
}

// tapped dismisses the software keyboard after a tap (a touch press and
// release in about the same place, not a scroll) outside the focused
// text field, as native apps do: the field loses focus. Not when focus
// moved since the press (the tap focused another field), nor on a
// release that ends a drag.
func (w *windowPointer) tapped(gtx layout.Context, e pointer.Event, owner elements.KeyboardOwner) {
	start := w.tapOwner
	w.tapOwner = nil
	if start == nil || e.Source != pointer.Touch || owner != start {
		return
	}
	if d := e.Position.Round().Sub(w.tapAt); d.X*d.X+d.Y*d.Y > gtx.Dp(12)*gtx.Dp(12) {
		return
	}
	gtx.Execute(key.FocusCmd{})
	gtx.Execute(key.SoftKeyboardCmd{Show: false})
}
