package dnd

import (
	"image"
	"time"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/semantic"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"gioui.org/unit"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements/box"
	"github.com/xmhrnx/giode/elements/internal/base"
)

// dragSlop is how far a mouse press, or a touch with TouchImmediate,
// moves before it becomes a drag.
const dragSlop = unit.Dp(6)

// SourceStyle is how a Draggable shows in its place while it's dragged.
type SourceStyle int

const (
	// SourceDim draws it faded (the default).
	SourceDim SourceStyle = iota
	// SourceHide leaves its space empty.
	SourceHide
	// SourceKeep draws it as it is.
	SourceKeep
)

// Axis limits the preview's movement.
type Axis int

const (
	// AxisBoth lets it follow the pointer anywhere (the default).
	AxisBoth Axis = iota
	// AxisVertical keeps it in its column, AxisHorizontal in its row.
	AxisVertical
	AxisHorizontal
)

// DragState is a Draggable's state, for its content to style itself.
type DragState struct {
	// Dragging is set while it's being dragged.
	Dragging bool
	// Hovered and Focused are the pointer over it, and keyboard focus.
	Hovered, Focused bool
}

// DragResult says how a drag ended.
type DragResult struct {
	// Dropped is set when it was dropped on a target that accepted it;
	// Target is that target and Index the list slot (-1 when the target
	// isn't a list).
	Dropped bool
	Target  *DropTarget
	Index   int
	// Cancelled is set when it ended without a drop: released where no
	// target accepts it, Escape, or Cancel.
	Cancelled bool
	// Exported is set when it left the window as an operating system
	// drag (Export); Dropped then says whether another application took
	// it.
	Exported bool
}

// Draggable is content that can be dragged to a DropTarget, carrying a
// payload. A mouse drag starts once the pointer moves a few dp with the
// button down; a touch drag starts with a long press (a haptic tap
// confirms it), so a finger moving at once still scrolls. While it's
// dragged, a preview follows the pointer: the content itself, or what
// Preview builds. Tab focuses it, and Space or Enter picks it up for a
// keyboard drag: arrows move it between drop slots, Space or Enter
// drops it, Escape cancels.
//
// Its content keeps its own clicks: a tap is not a drag. Create it once
// and reuse it every frame (for list items, look it up with Keyed).
type Draggable struct {
	content func(DragState) base.Element
	payload any
	preview func() base.Element
	source  SourceStyle
	axis    Axis
	label   string

	onStart func()
	onEnd   func(DragResult)
	export  func() ExportData

	disabled       bool
	touchImmediate bool

	origin image.Point
	size   image.Point
	// call is the content's recording, and callFrame the frame it's
	// from, for the preview.
	call      op.CallOp
	callFrame uint64

	hovered, focused bool
	pressed          bool
	pressID          pointer.ID
	pressPos         f32.Point
	pressAt          time.Time
	pressTouch       bool

	passFn func(gtx layout.Context) bool
}

// NewDraggable returns a Draggable carrying payload, drawing content.
func NewDraggable(payload any, content func(s DragState) base.Element) *Draggable {
	return &Draggable{payload: payload, content: content}
}

// Payload replaces the payload. It returns d for chaining.
func (d *Draggable) Payload(v any) *Draggable {
	d.payload = v
	return d
}

// Content replaces the content builder. It returns d for chaining.
func (d *Draggable) Content(fn func(s DragState) base.Element) *Draggable {
	d.content = fn
	return d
}

// Preview sets what follows the pointer, instead of the content itself.
// It's built when the drag starts. It returns d for chaining.
func (d *Draggable) Preview(fn func() base.Element) *Draggable {
	d.preview = fn
	return d
}

// Source sets how it shows in its place while dragged. It returns d for
// chaining.
func (d *Draggable) Source(s SourceStyle) *Draggable {
	d.source = s
	return d
}

// LockAxis keeps the preview moving along one axis only, as for
// reordering a list. It returns d for chaining.
func (d *Draggable) LockAxis(a Axis) *Draggable {
	d.axis = a
	return d
}

// TouchImmediate starts touch drags as soon as the finger moves, without
// a long press: for a drag handle, which isn't in the way of scrolling.
// It returns d for chaining.
func (d *Draggable) TouchImmediate(v bool) *Draggable {
	d.touchImmediate = v
	return d
}

// Label sets the name automation reports for it. It returns d for
// chaining.
func (d *Draggable) Label(s string) *Draggable {
	d.label = s
	return d
}

// Disabled turns dragging off. It returns d for chaining.
func (d *Draggable) Disabled(v bool) *Draggable {
	d.disabled = v
	if v {
		d.pressed = false
		if active != nil && active.src == d {
			Cancel()
		}
	}
	return d
}

// OnDragStart sets a handler for the start of a drag. It returns d for
// chaining.
func (d *Draggable) OnDragStart(fn func()) *Draggable {
	d.onStart = fn
	return d
}

// OnDragEnd sets a handler for the end of a drag, dropped or not. It
// runs after the target's OnDrop. It returns d for chaining.
func (d *Draggable) OnDragEnd(fn func(DragResult)) *Draggable {
	d.onEnd = fn
	return d
}

// Export makes it leave the window: dragged past the window's edge with
// a mouse, it becomes the operating system's drag of what fn returns -
// files or text - for other applications (a file manager, a mail, an
// editor, another copy of the app). The in-app drag ends there, and
// OnDragEnd runs when the system's drag does, with Exported set.
// Supported on Windows, macOS and Linux (CanExport); elsewhere the drag
// stays in the window. It returns d for chaining.
func (d *Draggable) Export(fn func() ExportData) *Draggable {
	d.export = fn
	return d
}

// Dragging reports whether it's being dragged.
func (d *Draggable) Dragging() bool { return active != nil && active.src == d }

// State returns its state.
func (d *Draggable) State() DragState {
	return DragState{Dragging: d.Dragging(), Hovered: d.hovered, Focused: d.focused}
}

// update processes its input; while it's dragged the window does
// (Drain).
func (d *Draggable) update(gtx layout.Context) (changed bool) {
	if d.disabled || d.Dragging() {
		return false
	}
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: d, Kinds: pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel | pointer.Enter | pointer.Leave})
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		switch e.Kind {
		case pointer.Enter:
			d.hovered, changed = true, true
		case pointer.Leave:
			d.hovered, changed = false, true
		case pointer.Cancel:
			d.pressed = false
			d.hovered = false
			changed = true
		case pointer.Press:
			if d.pressed || !e.Buttons.Contain(pointer.ButtonPrimary) && e.Source == pointer.Mouse {
				break
			}
			d.pressed = true
			d.pressID, d.pressPos, d.pressAt = e.PointerID, e.Position, gtx.Now
			d.pressTouch = e.Source == pointer.Touch
			if d.longPress() {
				kit.RequestFrameAt(gtx, gtx.Now.Add(box.LongPressDuration))
			}
		case pointer.Release:
			d.pressed = false
		case pointer.Drag:
			if !d.pressed || e.PointerID != d.pressID {
				break
			}
			slop := float32(gtx.Dp(dragSlop))
			delta := e.Position.Sub(d.pressPos)
			if delta.X*delta.X+delta.Y*delta.Y <= slop*slop {
				break
			}
			if d.longPress() {
				// Moved before the long press: a scroll, not a drag.
				d.pressed = false
				break
			}
			d.begin(gtx, e.Position)
			d.drainStart(gtx)
			return true
		}
	}
	if d.pressed && d.longPress() {
		if gtx.Now.Sub(d.pressAt) >= box.LongPressDuration {
			box.PlayLongPressFeedback()
			d.begin(gtx, d.pressPos)
			d.drainStart(gtx)
			return true
		}
		kit.RequestFrameAt(gtx, d.pressAt.Add(box.LongPressDuration))
	}
	for {
		ev, ok := gtx.Event(
			key.FocusFilter{Target: d},
			key.Filter{Focus: d, Name: key.NameSpace},
			key.Filter{Focus: d, Name: key.NameReturn},
			key.Filter{Focus: d, Name: key.NameEnter},
		)
		if !ok {
			break
		}
		switch e := ev.(type) {
		case key.FocusEvent:
			d.focused, changed = e.Focus, true
		case key.Event:
			if e.State == key.Press {
				d.beginKeyboard(gtx)
				return true
			}
		}
	}
	return changed
}

// longPress reports whether the press waits for a long press to drag.
func (d *Draggable) longPress() bool {
	return d.pressTouch && !d.touchImmediate
}

// begin starts a pointer drag; at is the pointer, relative to d.
func (d *Draggable) begin(gtx layout.Context, at f32.Point) {
	d.pressed = false
	o := f32.Pt(float32(d.origin.X), float32(d.origin.Y))
	grab := d.pressPos.Round()
	grab.X = max(0, min(grab.X, d.size.X))
	grab.Y = max(0, min(grab.Y, d.size.Y))
	start(d, &session{
		id: d.pressID, touch: d.pressTouch, pos: at.Add(o),
		grab: grab, size: d.size, srcOrigin: d.origin, started: gtx.Now,
	})
	// The drag is the window's: an enclosing Scroll doesn't scroll with
	// it, and the content's own clicks are cancelled.
	gtx.Execute(pointer.GrabCmd{Tag: d, ID: d.pressID})
	gtx.Execute(key.FocusCmd{Tag: d})
	gtx.Execute(op.InvalidateCmd{})
}

// drainStart reads the rest of the events queued for d when its drag
// started: they're relative to d, unlike the window's later ones.
func (d *Draggable) drainStart(gtx layout.Context) {
	s := active
	o := f32.Pt(float32(d.origin.X), float32(d.origin.Y))
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: d, Kinds: pointer.Drag | pointer.Release | pointer.Cancel})
		if !ok {
			return
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		switch e.Kind {
		case pointer.Drag:
			if e.PointerID == s.id {
				s.pos = e.Position.Add(o)
			}
		case pointer.Release:
			if e.PointerID == s.id {
				s.pos = e.Position.Add(o)
				s.release = true
			}
		case pointer.Cancel:
			s.cancel = true
		}
	}
}

// beginKeyboard picks it up from the keyboard.
func (d *Draggable) beginKeyboard(gtx layout.Context) {
	start(d, &session{
		keyboard: true, grab: d.size.Div(2), size: d.size, srcOrigin: d.origin,
		pos: f32.Pt(float32(d.origin.X+d.size.X/2), float32(d.origin.Y+d.size.Y/2)), started: gtx.Now,
	})
	gtx.Execute(op.InvalidateCmd{})
}

// pass is its input pass for enclosing caches.
func (d *Draggable) pass(gtx layout.Context) bool {
	return d.update(gtx)
}

// Layout lays it out.
func (d *Draggable) Layout(gtx layout.Context) layout.Dimensions {
	d.update(gtx)
	base.NoteDragAware()
	var el base.Element
	if d.content != nil {
		el = d.content(d.State())
	}
	base.TrackPosition(&d.origin)
	macro := op.Record(gtx.Ops)
	var dims layout.Dimensions
	if el != nil {
		dims = kit.Unselectable(el.Layout)(gtx)
	}
	call := macro.Stop()
	if !base.Measuring() {
		d.size = dims.Size
		d.call, d.callFrame = call, frame+1
	}

	dragging := d.Dragging()
	switch {
	case !dragging || d.source == SourceKeep:
		call.Add(gtx.Ops)
	case d.source == SourceDim:
		o := paint.PushOpacity(gtx.Ops, 0.35)
		call.Add(gtx.Ops)
		o.Pop()
	}

	// The hit area goes over the content, passing presses on: the
	// content keeps its clicks, and a drag cancels them.
	if !d.disabled {
		ps := pointer.PassOp{}.Push(gtx.Ops)
		cl := clip.Rect{Max: dims.Size}.Push(gtx.Ops)
		event.Op(gtx.Ops, d)
		kit.BlockDragHere(gtx)
		if d.label != "" {
			semantic.LabelOp(d.label).Add(gtx.Ops)
		}
		cl.Pop()
		ps.Pop()
	}
	if d.passFn == nil {
		d.passFn = d.pass
	}
	if !kit.HandleInput(d.passFn) && (d.pressed || d.focused || d.hovered) {
		kit.MarkLive()
	}
	if dragging || d.pressed {
		kit.MarkLive()
	}
	base.RecordSemantic(gtx, base.SemanticNode{Kind: base.SemanticButton, Label: d.label, Disabled: d.disabled, Target: d}, dims.Size)
	return dims
}
