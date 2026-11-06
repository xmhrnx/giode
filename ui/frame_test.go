package ui

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/elements"
)

// TestFrame checks Frame runs the steps components depend on: portals
// are drawn, after-layout callbacks run and ask for another frame, and
// State changes reach the invalidator.
func TestFrame(t *testing.T) {
	gtx := layout.Context{Ops: new(op.Ops), Constraints: layout.Exact(image.Pt(640, 480))}
	drawn := false
	view := Raw(func(gtx layout.Context) layout.Dimensions {
		if elements.WindowSize() != image.Pt(640, 480) {
			t.Errorf("window size %v during layout", elements.WindowSize())
		}
		elements.RegisterPortal(gtx, image.Pt(10, 10), func() Element {
			return Raw(func(gtx layout.Context) layout.Dimensions {
				drawn = true
				return layout.Dimensions{Size: image.Pt(5, 5)}
			})
		})
		return layout.Dimensions{Size: gtx.Constraints.Max}
	})
	if Frame(gtx, view) {
		t.Error("a plain frame asked for another")
	}
	if !drawn {
		t.Error("the portal wasn't drawn")
	}

	woken := 0
	SetInvalidator(func() { woken++ })
	defer SetInvalidator(nil)
	s := NewState(1)
	s.Set(2)
	if woken == 0 {
		t.Error("State.Set didn't wake the host")
	}
}

// A release on empty space survives Gio replaying the frame's events
// when a command runs (see keepReleases): the next press still reaches
// the widget.
func TestFrameKeepsReleaseThroughReplay(t *testing.T) {
	var r input.Router
	button, area := new(int), new(int)
	presses := 0
	view := Raw(func(gtx layout.Context) layout.Dimensions {
		for {
			ev, ok := gtx.Event(pointer.Filter{Target: button, Kinds: pointer.Press | pointer.Release})
			if !ok {
				break
			}
			if e, ok := ev.(pointer.Event); ok && e.Kind == pointer.Press {
				presses++
			}
		}
		cl := clip.Rect{Min: image.Pt(200, 0), Max: image.Pt(400, 200)}.Push(gtx.Ops)
		event.Op(gtx.Ops, button)
		cl.Pop()
		// A window-wide, pass-through listener that takes only presses,
		// as menu.Context's right-click area does.
		for {
			if _, ok := gtx.Event(pointer.Filter{Target: area, Kinds: pointer.Press}); !ok {
				break
			}
		}
		ps := pointer.PassOp{}.Push(gtx.Ops)
		cl = clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
		event.Op(gtx.Ops, area)
		cl.Pop()
		ps.Pop()
		return layout.Dimensions{Size: gtx.Constraints.Max}
	})
	frame := func(blink bool) {
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Source: r.Source(), Constraints: layout.Exact(image.Pt(400, 200)), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
		if blink {
			gtx.Execute(op.InvalidateCmd{})
		}
		Frame(gtx, view)
		r.Frame(&ops)
	}
	frame(false)
	frame(false)
	r.Queue(
		pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: f32.Pt(50, 50)},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: f32.Pt(50, 50)},
	)
	frame(true)
	frame(false)
	r.Queue(
		pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: f32.Pt(300, 50)},
		pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: f32.Pt(300, 50)},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: f32.Pt(300, 50)},
	)
	frame(false)
	if presses != 1 {
		t.Errorf("the widget got %d presses after a release on empty space, want 1", presses)
	}
}
