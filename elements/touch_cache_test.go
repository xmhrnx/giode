package elements

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
)

// Switching to touch input changes minimum touch targets, so a cached
// subtree must be laid out again.
func TestMemoRebuildsOnTouchInput(t *testing.T) {
	defer SetTouchInput(TouchInput())
	SetTouchInput(false)
	frame := func() layout.Dimensions {
		BeginFrame()
		defer EndFrame()
		var ops op.Ops
		return Memo("target", func() Element {
			return Raw(func(gtx layout.Context) layout.Dimensions {
				return MinTouchTarget(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Dimensions{Size: image.Pt(10, 10)}
				})
			})
		}).Layout(layout.Context{Ops: &ops, Constraints: layout.Constraints{Max: image.Pt(200, 200)}})
	}
	if s := frame().Size; s != image.Pt(10, 10) {
		t.Fatalf("mouse size %v", s)
	}
	SetTouchInput(true)
	if s := frame().Size; s.X < 40 {
		t.Errorf("after switching to touch: %v, want the 48dp target", s)
	}
}
