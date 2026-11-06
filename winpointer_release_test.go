package giode

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
)

// A release on empty space - where only the window-wide listener is -
// survives Gio replaying the frame's events when a command runs (a
// caret blink's InvalidateCmd): otherwise the replay restores the press
// without its release, the button stays down as far as Gio knows, and
// the next press reaches only the window listener.
func TestWindowPointerKeepsReleaseThroughReplay(t *testing.T) {
	var r input.Router
	a := &App{}
	var wp windowPointer
	button := new(int)
	var presses int
	frame := func(blink bool) {
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Source: r.Source(), Constraints: layout.Exact(image.Pt(400, 200))}
		if blink {
			gtx.Execute(op.InvalidateCmd{})
		}
		// The button: the right half.
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
		wp.layout(a, gtx)
		r.Frame(&ops)
	}
	frame(false)
	frame(false)
	// Press and release on empty space, then a frame that blinks.
	r.Queue(
		pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: f32.Pt(50, 50)},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: f32.Pt(50, 50)},
	)
	frame(true)
	frame(false)
	// Now a click on the button.
	r.Queue(
		pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: f32.Pt(300, 50)},
		pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: f32.Pt(300, 50)},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: f32.Pt(300, 50)},
	)
	frame(false)
	if presses != 1 {
		t.Errorf("the button got %d presses after a release on empty space, want 1", presses)
	}
}
