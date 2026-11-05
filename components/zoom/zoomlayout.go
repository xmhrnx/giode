// Package zoom provides Zoom, a view that pinch-zooms and pans its
// content: a page viewer, a photo, a crop editor.

package zoom

import (
	"image"
	"time"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
)

// Layout lays out the content, scaled and moved, clipped to the view.
func (z *Zoom) Layout(gtx layout.Context) layout.Dimensions {
	view := gtx.Constraints.Max
	if view.X >= 1<<24 || view.Y >= 1<<24 {
		// Unbounded: take the content's size.
		view = image.Point{}
	}
	cgtx := gtx
	cgtx.Constraints.Min = image.Point{}
	if view != (image.Point{}) {
		cgtx.Constraints.Max = view
	}
	// Measure the content at its fitted size; a change of size (the
	// window turned) refits it.
	size := elements.Measure(cgtx, z.content()).Size
	if view == (image.Point{}) {
		view = size
	}
	refit := size != z.size || view != z.view
	z.size, z.view = size, view
	if refit {
		z.scale = clampf(z.scale, z.minScale, z.maxScale)
		z.offset = z.clampOffset(z.scale, z.offset)
	}

	z.update(gtx)
	z.animate(gtx)

	defer clip.Rect{Max: view}.Push(gtx.Ops).Pop()
	// Zoom's own input sits under the content, which gets first pick.
	event.Op(gtx.Ops, z)

	tr := f32.Affine2D{}.Scale(f32.Point{}, f32.Pt(z.scale, z.scale)).Offset(z.offset)
	t := op.Affine(tr).Push(gtx.Ops)
	z.content().Layout(cgtx)
	t.Pop()

	if cur := (state{z.scale, z.offset}); cur != z.lastChanged {
		z.lastChanged = cur
		if z.onChange != nil {
			z.onChange(z.scale, z.offset)
		}
	}
	return layout.Dimensions{Size: view}
}

// target is where the view is heading: the end of a running animation,
// else where it is.
func (z *Zoom) target() state {
	if z.animating {
		return z.animTo
	}
	return state{z.scale, z.offset}
}

// canPan reports whether one finger may move the content: when it's
// larger than the view.
func (z *Zoom) canPan() bool {
	return float32(z.size.X)*z.scale > float32(z.view.X)+0.5 || float32(z.size.Y)*z.scale > float32(z.view.Y)+0.5
}

// panRange is how far the wheel may move the content each way.
func (z *Zoom) panRange() (x, y pointer.ScrollRange) {
	lo := z.clampOffset(z.scale, f32.Pt(-1e9, -1e9))
	hi := z.clampOffset(z.scale, f32.Pt(1e9, 1e9))
	// Scrolling by +d moves the content by -d.
	x = pointer.ScrollRange{Min: int(z.offset.X - hi.X), Max: int(z.offset.X - lo.X)}
	y = pointer.ScrollRange{Min: int(z.offset.Y - hi.Y), Max: int(z.offset.Y - lo.Y)}
	return x, y
}

// zoomAround zooms to scale keeping the content point under p (a view
// point) where it is.
func (z *Zoom) zoomAround(scale float32, p f32.Point, animate bool) {
	from := z.target()
	q := p.Sub(from.offset).Div(from.scale)
	to := state{scale: scale, offset: z.clampOffset(scale, p.Sub(q.Mul(scale)))}
	if !animate {
		z.scale, z.offset = to.scale, to.offset
		return
	}
	z.animFrom, z.animTo = state{z.scale, z.offset}, to
	z.animStart = time.Time{}
	z.animating = true
}

// animate steps a running animation.
func (z *Zoom) animate(gtx layout.Context) {
	if !z.animating {
		return
	}
	if z.animStart.IsZero() {
		z.animStart = gtx.Now
	}
	t := float32(gtx.Now.Sub(z.animStart)) / float32(animDuration)
	if t >= 1 {
		z.scale, z.offset = z.animTo.scale, z.animTo.offset
		z.animating = false
		return
	}
	t = 1 - (1-t)*(1-t)*(1-t) // ease out
	z.scale = z.animFrom.scale + (z.animTo.scale-z.animFrom.scale)*t
	z.offset = z.animFrom.offset.Add(z.animTo.offset.Sub(z.animFrom.offset).Mul(t))
	kit.RequestFrame(gtx)
}

// clampOffset keeps the content covering the view where it's larger,
// and centred where it's smaller.
func (z *Zoom) clampOffset(scale float32, off f32.Point) f32.Point {
	w, h := float32(z.size.X)*scale, float32(z.size.Y)*scale
	vw, vh := float32(z.view.X), float32(z.view.Y)
	clampAxis := func(o, s, v float32) float32 {
		if s <= v {
			return (v - s) / 2
		}
		return clampf(o, v-s, 0)
	}
	return f32.Pt(clampAxis(off.X, w, vw), clampAxis(off.Y, h, vh))
}

func clampf(v, lo, hi float32) float32 { return max(lo, min(v, hi)) }

func sq(v float32) float32 { return v * v }
