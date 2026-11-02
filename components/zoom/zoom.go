// Package zoom provides Zoom, a view that pinch-zooms and pans its
// content: a page viewer, a photo, a crop editor.
package zoom

import (
	"image"
	"math"
	"time"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
)

// Zoom shows its content scaled and moved by the user's gestures:
//
//   - two fingers pinch to zoom around their midpoint, and move the
//     content as they move;
//   - one finger pans the content once it's zoomed in;
//   - a double tap zooms in at the tap (to DoubleTapScale) or back out;
//   - on desktops, Ctrl (Cmd on macOS) with the wheel or a trackpad
//     zooms at the pointer, and the wheel pans a zoomed view.
//
// Zoom fills the space it's given and centres the content, laid out to
// fit it, at scale 1. Gestures the content handles itself (a crop
// handle's drag, a button) stay with it: Zoom sees only the touches
// and clicks the content doesn't take, and the content's own input is
// transformed with it, so its handlers keep working at any scale.
//
// Create it once and lay it out every frame. Its methods are for the UI
// goroutine.
type Zoom struct {
	content func() elements.Element

	minScale, maxScale, doubleTap float32
	wheelZoom                     bool
	onChange                      func(scale float32, offset f32.Point)

	scale  float32
	offset f32.Point // the content's top left, in viewport pixels
	view   image.Point
	size   image.Point // the content's unscaled size

	// Active touches by pointer ID, and the gesture they started from.
	touches map[pointer.ID]f32.Point
	start   gestureStart
	panning bool

	// The last tap, to recognise a double tap.
	lastTap     time.Time
	lastTapPos  f32.Point
	pressAt     f32.Point
	pressMoved  bool
	animFrom    state
	animTo      state
	animStart   time.Time
	animating   bool
	lastChanged state
}

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

// New returns a Zoom over content, built each frame. It zooms between 1
// and 5 times, and a double tap zooms to 2.5.
func New(content func() elements.Element) *Zoom {
	return &Zoom{content: content, minScale: 1, maxScale: 5, doubleTap: 2.5, scale: 1}
}

// Limits sets the smallest and largest scale. A minimum below 1 lets the
// content shrink below its fitted size. It returns z for chaining.
func (z *Zoom) Limits(min, max float32) *Zoom {
	if min > 0 && max >= min {
		z.minScale, z.maxScale = min, max
		z.scale = clampf(z.scale, min, max)
	}
	return z
}

// DoubleTapScale sets the scale a double tap zooms to; 0 turns double
// taps off. It returns z for chaining.
func (z *Zoom) DoubleTapScale(s float32) *Zoom {
	z.doubleTap = s
	return z
}

// WheelZoom makes the mouse wheel zoom without a modifier key, as
// image viewers do, instead of panning. Leave it off for a Zoom inside
// a scrolling list, where the wheel should scroll the list. It returns
// z for chaining.
func (z *Zoom) WheelZoom(on bool) *Zoom {
	z.wheelZoom = on
	return z
}

// OnChange sets a function called when the scale or offset changes,
// with the new values. It returns z for chaining.
func (z *Zoom) OnChange(fn func(scale float32, offset f32.Point)) *Zoom {
	z.onChange = fn
	return z
}

// Scale returns the current scale.
func (z *Zoom) Scale() float32 { return z.scale }

// Offset returns where the content's top left corner sits in the view,
// in pixels; negative when it's zoomed past the edges.
func (z *Zoom) Offset() f32.Point { return z.offset }

// Zoomed reports whether the content is zoomed in past its fitted size.
func (z *Zoom) Zoomed() bool { return z.scale > 1.001 }

// SetScale zooms to s around the view's centre, animated.
func (z *Zoom) SetScale(s float32) {
	z.zoomAround(clampf(s, z.minScale, z.maxScale), f32.Pt(float32(z.view.X)/2, float32(z.view.Y)/2), true)
}

// ZoomIn zooms in by a step (1.25x) around the centre, for a button or
// a shortcut.
func (z *Zoom) ZoomIn() { z.SetScale(z.target().scale * 1.25) }

// ZoomOut zooms out by a step.
func (z *Zoom) ZoomOut() { z.SetScale(z.target().scale / 1.25) }

// Reset returns to scale 1, centred, animated.
func (z *Zoom) Reset() { z.SetScale(1) }

// ToView converts a point in the content's own (unscaled) coordinates to
// the view's.
func (z *Zoom) ToView(p f32.Point) f32.Point { return z.offset.Add(p.Mul(z.scale)) }

// ToContent converts a point in the view to the content's own
// coordinates: where a tap on the view lands on the page.
func (z *Zoom) ToContent(p f32.Point) f32.Point { return p.Sub(z.offset).Div(z.scale) }

// target is where the view is heading: the end of a running animation,
// else where it is.
func (z *Zoom) target() state {
	if z.animating {
		return z.animTo
	}
	return state{z.scale, z.offset}
}

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
