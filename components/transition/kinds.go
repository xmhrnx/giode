// Package transition animates a component in and out: fade, slide,
// enlarge, shrink, rotate, collapse - alone or combined - with a
// configurable duration and CSS-style easing.

package transition

import (
	"time"

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
