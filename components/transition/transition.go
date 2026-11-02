// Package transition animates a component in and out: fade, slide,
// enlarge, shrink, rotate, collapse - alone or combined - with a
// configurable duration and CSS-style easing.
package transition

import (
	"math"
	"time"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles/properties"
)

// Kind is a transition effect. Kinds combine with |:
//
//	transition.Fade | transition.SlideUp
type Kind uint

const (
	// Fade animates opacity from 0.
	Fade Kind = 1 << iota
	// SlideUp enters moving up from below its place.
	SlideUp
	// SlideDown enters moving down from above.
	SlideDown
	// SlideLeft enters moving left, from its right.
	SlideLeft
	// SlideRight enters moving right, from its left.
	SlideRight
	// Enlarge grows from small to full size.
	Enlarge
	// Shrink settles from larger than full size.
	Shrink
	// Rotate turns in from a quarter turn.
	Rotate
	// Collapse animates the height it takes in the layout, pushing
	// the content below it, with the content clipped - for expanding
	// panels and accordions.
	Collapse
	// CollapseWidth is Collapse along the horizontal axis - for side
	// panels.
	CollapseWidth
)

// Common combinations.
const (
	// FadeUp fades in while rising - a good default for content
	// appearing.
	FadeUp = Fade | SlideUp
	// Zoom fades in while growing - for dialogs and popovers.
	Zoom = Fade | Enlarge
	// Pop fades in while shrinking to size.
	Pop = Fade | Shrink
)

// Speed presets for the duration argument.
const (
	Fast   = 150 * time.Millisecond
	Normal = 250 * time.Millisecond
	Slow   = 500 * time.Millisecond
)

// Easing is a CSS-style timing function; see properties.Easing for the
// built-ins (EaseOut, EaseInOut, EaseOutBack, ...) and custom
// "cubic-bezier(x1, y1, x2, y2)" values.
type Easing = properties.Easing

// Effect tuning: how far slides travel (as a fraction of the
// component's size, capped), and where scale and rotation start.
const (
	slideFraction = 0.25
	maxSlideDp    = 48
	enlargeFrom   = 0.6
	shrinkFrom    = 1.25
	rotateFromDeg = -90
)

// Transition animates a component in and out. Create it once with New
// and lay it out every frame; Show, Hide and Toggle start the
// animation, which reverses smoothly if interrupted:
//
//	panel := giode.Transition(content, giode.FadeUp, giode.TransitionNormal, giode.EaseOut)
//	toggle := giode.Button("Details").OnClick(panel.Toggle)
//
// A hidden transition takes no space and lays nothing out.
type Transition struct {
	child    elements.Element
	kind     Kind
	duration time.Duration
	easing   Easing

	visible   bool
	progress  float32 // 0 hidden, 1 shown
	animating bool
	lastFrame time.Time
	appear    bool
	onDone    func(visible bool)
}

// New returns a transition for child, initially visible and animating
// in on its first layout. kind selects the effect (combine with |),
// duration the speed (Fast, Normal, Slow or any duration; zero means
// Normal) and easing the timing function ("" means EaseInOut).
func New(child elements.Element, kind Kind, duration time.Duration, easing Easing) *Transition {
	return &Transition{child: child, kind: kind, duration: duration, easing: easing, visible: true, appear: true}
}

// Child replaces the animated component. It returns t for chaining.
func (t *Transition) Child(child elements.Element) *Transition {
	t.child = child
	return t
}

// Kind changes the effect. It returns t for chaining.
func (t *Transition) Kind(k Kind) *Transition {
	t.kind = k
	return t
}

// Duration changes the speed. It returns t for chaining.
func (t *Transition) Duration(d time.Duration) *Transition {
	t.duration = d
	return t
}

// Easing changes the timing function. It returns t for chaining.
func (t *Transition) Easing(e Easing) *Transition {
	t.easing = e
	return t
}

// Hidden starts the transition hidden instead of animating in on first
// layout; call Show to bring it in. It returns t for chaining.
func (t *Transition) Hidden() *Transition {
	t.visible, t.progress, t.appear = false, 0, false
	return t
}

// Appear controls whether a transition created visible animates in on
// its first layout (the default) or starts fully shown. It returns t
// for chaining.
func (t *Transition) Appear(animate bool) *Transition {
	t.appear = animate
	if !animate && t.visible {
		t.progress = 1
	}
	return t
}

// OnDone sets a handler called when an animation finishes, with the
// final visibility. It returns t for chaining.
func (t *Transition) OnDone(fn func(visible bool)) *Transition {
	t.onDone = fn
	return t
}

// Show animates the component in.
func (t *Transition) Show() { t.SetVisible(true) }

// Hide animates the component out.
func (t *Transition) Hide() { t.SetVisible(false) }

// Toggle animates the component in or out.
func (t *Transition) Toggle() { t.SetVisible(!t.visible) }

// SetVisible animates to the given visibility.
func (t *Transition) SetVisible(v bool) {
	if t.visible == v {
		return
	}
	t.visible = v
	elements.InvalidateCaches()
}

// Visible reports the visibility being animated to.
func (t *Transition) Visible() bool { return t.visible }

// Animating reports whether an animation is in progress.
func (t *Transition) Animating() bool {
	target := float32(0)
	if t.visible {
		target = 1
	}
	return t.progress != target
}

// Progress returns the animation position: 0 fully hidden, 1 fully
// shown, before easing.
func (t *Transition) Progress() float32 { return t.progress }

// advance moves the animation to gtx.Now.
func (t *Transition) advance(gtx layout.Context) {
	target := float32(0)
	if t.visible {
		target = 1
	}
	if t.progress == target {
		t.animating = false
		t.lastFrame = gtx.Now
		return
	}
	// An animation starting after the component sat idle measures
	// from this frame, not from the last frame drawn.
	if !t.animating || t.lastFrame.IsZero() {
		t.lastFrame = gtx.Now
	}
	t.animating = true
	d := t.duration
	if d <= 0 {
		d = Normal
	}
	step := float32(gtx.Now.Sub(t.lastFrame)) / float32(d)
	t.lastFrame = gtx.Now
	if t.progress < target {
		t.progress = min(target, t.progress+step)
	} else {
		t.progress = max(target, t.progress-step)
	}
	if t.progress == target {
		t.animating = false
		if t.onDone != nil {
			t.onDone(t.visible)
		}
	}
	// Keep the frames coming, including the one that paints the final
	// state.
	kit.RequestFrame(gtx)
}

// Layout lays out and animates the component.
func (t *Transition) Layout(gtx layout.Context) layout.Dimensions {
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

func clamp01(v float32) float32 {
	return max(0, min(1, v))
}
