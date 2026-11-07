// Package transition animates a component in and out: fade, slide,
// enlarge, shrink, rotate, collapse - alone or combined - with a
// configurable duration and CSS-style easing.

package transition

import (
	"math"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// Layout lays out and animates the component.
func (t *Transition) Layout(gtx layout.Context) layout.Dimensions {
	dims := t.layoutSelf(gtx)
	// A right-click or long-press anywhere on it opens its context menu.
	t.AnchorContext(gtx, dims.Size)
	return dims
}

// layoutSelf lays it out without its context menu area.
func (t *Transition) layoutSelf(gtx layout.Context) layout.Dimensions {
	t.advance(gtx)
	if t.progress <= 0 && !t.visible || t.child == nil {
		return layout.Dimensions{}
	}
	e := t.easing.Apply(t.progress)

	// Lay the child out once, recorded, to know its size.
	macro := op.Record(gtx.Ops)
	dims := t.child.Layout(gtx)
	call := macro.Stop()
	size := dims.Size

	// Collapse animates the space taken in the layout.
	out := dims
	collapse := t.kind&(Collapse|CollapseWidth) != 0
	if t.kind&Collapse != 0 {
		out.Size.Y = int(float32(size.Y)*clamp01(e) + 0.5)
	}
	if t.kind&CollapseWidth != 0 {
		out.Size.X = int(float32(size.X)*clamp01(e) + 0.5)
	}
	if collapse {
		defer clip.Rect{Max: out.Size}.Push(gtx.Ops).Pop()
	}

	if t.kind&Fade != 0 {
		defer paint.PushOpacity(gtx.Ops, clamp01(e)).Pop()
	}

	// Offsets, scale and rotation, all relative to the child's center.
	rest := 1 - e
	var off f32.Point
	slide := func(extent int) float32 {
		return min(float32(extent)*slideFraction, float32(gtx.Dp(maxSlideDp))) * rest
	}
	if t.kind&SlideUp != 0 {
		off.Y += slide(size.Y)
	}
	if t.kind&SlideDown != 0 {
		off.Y -= slide(size.Y)
	}
	if t.kind&SlideLeft != 0 {
		off.X += slide(size.X)
	}
	if t.kind&SlideRight != 0 {
		off.X -= slide(size.X)
	}
	scale := float32(1)
	if t.kind&Enlarge != 0 {
		scale *= enlargeFrom + (1-enlargeFrom)*e
	}
	if t.kind&Shrink != 0 {
		scale *= shrinkFrom + (1-shrinkFrom)*e
	}
	angle := float32(0)
	if t.kind&Rotate != 0 {
		angle = rotateFromDeg * rest * math.Pi / 180
	}
	if off != (f32.Point{}) || scale != 1 || angle != 0 {
		c := f32.Pt(float32(size.X)/2, float32(size.Y)/2)
		if scale < 0 {
			scale = 0
		}
		a := f32.AffineId().
			Scale(c, f32.Pt(scale, scale)).
			Rotate(c, angle).
			Offset(off)
		defer op.Affine(a).Push(gtx.Ops).Pop()
	}
	call.Add(gtx.Ops)
	return out
}
