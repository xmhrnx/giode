// Package zoom provides Zoom, a view that pinch-zooms and pans its
// content: a page viewer, a photo, a crop editor.

package zoom

import (
	"math"
	"time"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"github.com/xmhrnx/giode/components/kit"
)

// gestureStart is what a pinch or pan is measured from.
type gestureStart struct {
	scale    float32
	offset   f32.Point
	centroid f32.Point
	dist     float32
}

type state struct {
	scale  float32
	offset f32.Point
}

const animDuration = 200 * time.Millisecond

// update handles touches, clicks and the wheel.
func (z *Zoom) update(gtx layout.Context) {
	// The wheel pans a zoomed view within its edges, so a list around
	// it scrolls once it can't; with WheelZoom (or a modifier, which only
	// arrives while Zoom takes the wheel) it takes it all.
	rx, ry := z.panRange()
	if z.wheelZoom {
		ry = pointer.ScrollRange{Min: math.MinInt32, Max: math.MaxInt32}
	}
	for {
		ev, ok := gtx.Source.Event(pointer.Filter{
			Target:  z,
			Kinds:   pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel | pointer.Scroll,
			ScrollX: rx,
			ScrollY: ry,
		})
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		switch e.Kind {
		case pointer.Press:
			z.press(gtx, e)
		case pointer.Drag:
			z.drag(gtx, e)
		case pointer.Release, pointer.Cancel:
			z.release(gtx, e)
		case pointer.Scroll:
			z.wheel(gtx, e)
		}
	}
}

func (z *Zoom) press(gtx layout.Context, e pointer.Event) {
	if z.touches == nil {
		z.touches = map[pointer.ID]f32.Point{}
	}
	z.animating = false
	z.touches[e.PointerID] = e.Position
	z.restart()
	if len(z.touches) == 1 {
		z.pressAt, z.pressMoved = e.Position, false
	}
}

func (z *Zoom) drag(gtx layout.Context, e pointer.Event) {
	if _, ok := z.touches[e.PointerID]; !ok {
		return
	}
	z.touches[e.PointerID] = e.Position
	slop := float32(gtx.Dp(8))
	if sq(e.Position.X-z.pressAt.X)+sq(e.Position.Y-z.pressAt.Y) > slop*slop {
		z.pressMoved = true
	}
	c, d := z.centroid()
	switch {
	case len(z.touches) >= 2:
		scale := clampf(z.start.scale*d/max(z.start.dist, 1), z.minScale, z.maxScale)
		// Keep the content point that was under the fingers' midpoint
		// under it, wherever the midpoint moves.
		q := z.start.centroid.Sub(z.start.offset).Div(z.start.scale)
		z.scale = scale
		z.offset = z.clampOffset(scale, c.Sub(q.Mul(scale)))
		z.panning = true
		gtx.Execute(pointer.GrabCmd{Tag: z, ID: e.PointerID})
	case z.canPan() && (z.panning || z.pressMoved):
		z.panning = true
		z.offset = z.clampOffset(z.scale, z.start.offset.Add(c.Sub(z.start.centroid)))
		gtx.Execute(pointer.GrabCmd{Tag: z, ID: e.PointerID})
	}
	kit.RequestFrame(gtx)
}

func (z *Zoom) release(gtx layout.Context, e pointer.Event) {
	if _, ok := z.touches[e.PointerID]; !ok {
		return
	}
	delete(z.touches, e.PointerID)
	z.restart()
	if len(z.touches) > 0 {
		return
	}
	wasPanning := z.panning
	z.panning = false
	if e.Kind == pointer.Cancel || wasPanning || z.pressMoved || z.doubleTap <= 0 {
		return
	}
	// A tap: the second within a short time and distance zooms.
	near := float32(gtx.Dp(32))
	if !z.lastTap.IsZero() && gtx.Now.Sub(z.lastTap) < 350*time.Millisecond &&
		sq(e.Position.X-z.lastTapPos.X)+sq(e.Position.Y-z.lastTapPos.Y) < near*near {
		z.lastTap = time.Time{}
		to := z.doubleTap
		if z.target().scale > 1.001 {
			to = 1
		}
		z.zoomAround(clampf(to, z.minScale, z.maxScale), e.Position, true)
		kit.RequestFrame(gtx)
		return
	}
	z.lastTap, z.lastTapPos = gtx.Now, e.Position
}

func (z *Zoom) wheel(gtx layout.Context, e pointer.Event) {
	if e.Modifiers.Contain(key.ModShortcut) || z.wheelZoom {
		// About 10% per wheel notch (Gio reports ~ 50-100 px per notch
		// as scroll distance).
		f := float32(math.Exp(float64(-e.Scroll.Y) / 400))
		z.animating = false
		z.zoomAround(clampf(z.scale*f, z.minScale, z.maxScale), e.Position, false)
	} else {
		z.offset = z.clampOffset(z.scale, z.offset.Sub(e.Scroll))
	}
	kit.RequestFrame(gtx)
}

// restart measures the gesture from the touches as they are now, so a
// finger added or lifted doesn't make the content jump.
func (z *Zoom) restart() {
	c, d := z.centroid()
	z.start = gestureStart{scale: z.scale, offset: z.offset, centroid: c, dist: d}
}

// centroid returns the touches' midpoint and their mean distance from it.
func (z *Zoom) centroid() (f32.Point, float32) {
	if len(z.touches) == 0 {
		return f32.Point{}, 0
	}
	var c f32.Point
	for _, p := range z.touches {
		c = c.Add(p)
	}
	c = c.Div(float32(len(z.touches)))
	var d float32
	for _, p := range z.touches {
		d += float32(math.Hypot(float64(p.X-c.X), float64(p.Y-c.Y)))
	}
	return c, d / float32(len(z.touches))
}
