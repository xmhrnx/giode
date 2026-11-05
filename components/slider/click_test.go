package slider

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
)

// TestClickSeeks checks that a click on the track, without dragging,
// moves the value there and reports it.
func TestClickSeeks(t *testing.T) {
	var r input.Router
	s := New()
	var got float32 = -1
	s.OnChange(func(v float32) { got = v })
	frame := func() {
		ops := new(op.Ops)
		gtx := layout.Context{Ops: ops, Source: r.Source(), Constraints: layout.Exact(image.Pt(200, 24))}
		s.Layout(gtx)
		r.Frame(ops)
	}
	frame()
	r.Queue(
		pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: f32.Pt(150, 12)},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: f32.Pt(150, 12)},
	)
	frame()
	if got < 0.7 || got > 0.8 {
		t.Fatalf("after click at 75%% of the track: OnChange got %v, value %v; want about 0.75", got, s.Value())
	}
}
