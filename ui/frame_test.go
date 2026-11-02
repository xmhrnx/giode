package ui

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"

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
