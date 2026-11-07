// Package transition animates a component in and out: fade, slide,
// enlarge, shrink, rotate, collapse - alone or combined - with a
// configurable duration and CSS-style easing.

package transition

import (
	"time"

	"github.com/xmhrnx/giode/elements"
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
	// Contextual gives it a context menu (Context).
	elements.Contextual

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
