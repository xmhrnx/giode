package dialog

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
)

// TestClickInsidePanelKeepsOpen checks that clicking the panel's empty
// space doesn't reach the scrim, while clicking outside the panel still
// closes the dialog.
func TestClickInsidePanelKeepsOpen(t *testing.T) {
	open := true
	d := New(&open, func() elements.Element {
		return elements.Box(styles.Styles{Width: 200, Height: 100})
	})
	var r input.Router
	win := image.Pt(800, 600)
	// The dialog paints as an overlay: drain the queue as the app does
	// each frame, or neither the scrim nor the panel is laid out.
	elements.SetWindowSize(win)
	defer elements.SetWindowSize(image.Point{})
	frame := func() {
		ops := new(op.Ops)
		gtx := layout.Context{Ops: ops, Source: r.Source(), Constraints: layout.Exact(win)}
		d.Layout(gtx)
		elements.DrainPortals(gtx)
		r.Frame(ops)
	}
	click := func(x, y float32) {
		r.Queue(
			pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: f32.Pt(x, y)},
			pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: f32.Pt(x, y)},
		)
		frame()
		frame()
	}
	frame()
	frame()
	click(400, 300) // the panel's center
	if !open {
		t.Fatal("clicking inside the panel closed the dialog")
	}
	click(20, 20)
	if open {
		t.Fatal("clicking the scrim did not close the dialog")
	}
}
