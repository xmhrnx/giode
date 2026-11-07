// Package zoom provides Zoom, a view that pinch-zooms and pans its
// content: a page viewer, a photo, a crop editor.

package zoom

import (
	"image"
	"time"

	"gioui.org/f32"
	"gioui.org/io/pointer"
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
	// Contextual gives it a context menu (Context).
	elements.Contextual

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
