// Package drawer provides a stateful slide-out side panel.
package drawer

import (
	"gioui.org/layout"
	"github.com/xmhrnx/giode/components/kit"
	"time"

	"gioui.org/gesture"
	"gioui.org/widget"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
)

// Side selects which edge the drawer slides from.
type Side int

const (
	// Left slides in from the left edge.
	Left Side = iota
	// Right slides in from the right edge.
	Right
	// Start slides in from the edge where lines start: left, or right
	// in a right-to-left locale.
	Start
	// End slides in from the edge where lines end.
	End
)

// edge resolves Start and End to the physical edge for gtx.
func (d *Drawer) edge(gtx layout.Context) Side {
	switch d.side {
	case Start:
		if kit.RTL(gtx) {
			return Right
		}
		return Left
	case End:
		if kit.RTL(gtx) {
			return Left
		}
		return Right
	}
	return d.side
}

var defaultBackground = "#1e293b"

// Drawer is a side panel with a scrim, sliding in over 200ms. It
// implements elements.Element and can be laid out anywhere in the
// tree: it takes no space, and paints over the whole window as an
// overlay (see elements.RegisterOverlay) while open. Drag the panel
// toward its edge to close it; with SwipeToOpen, drag from the edge to
// open it.
//
// Create it once with New, outside the view function.
type Drawer struct {
	open    *bool
	wasOpen bool
	side    Side
	width   int
	content func() elements.Element
	st      styles.Styles
	// fixed disables closing from the scrim, Back, Escape and dragging.
	fixed bool
	// edgeSwipe enables dragging the drawer open from its edge.
	edgeSwipe bool

	// progress is how far open the drawer is, 0 to 1, animated toward
	// *open, and last when it was advanced.
	progress float32
	last     time.Time
	// duration is the slide's length; 0 shows and hides the drawer at
	// once (see Duration).
	duration time.Duration

	scrim widget.Clickable
	// drag follows a finger or mouse moving the panel (or, while
	// closed, opening it from the edge).
	drag      gesture.Drag
	dragFrom  float32
	dragStart float32
	dragging  bool
}

// New returns a Drawer bound to open: it slides in from side whenever
// *open is true, and slides back out when *open becomes false
// (including from clicking the scrim). content builds the panel body
// fresh every frame the drawer is shown, so it can reflect state that
// changed since the drawer opened. The styles argument is optional.
func New(open *bool, side Side, width int, content func() elements.Element, st ...styles.Styles) *Drawer {
	d := &Drawer{open: open, side: side, width: width, content: content, duration: animDuration}
	if len(st) > 0 {
		d.st = st[0]
	}
	return d
}

// Opened reports whether the drawer is currently open (it may still
// be mid slide-out animation after *open turns false).
func (d *Drawer) Opened() bool {
	return d.open != nil && *d.open
}

// Duration sets how long the drawer takes to slide in or out (200ms by
// default); 0 shows and hides it at once. It returns d for chaining.
func (d *Drawer) Duration(t time.Duration) *Drawer {
	d.duration = max(0, t)
	return d
}

// Animated turns the slide on (the default) or off: Animated(false)
// shows and hides the drawer at once, like Duration(0). It returns d
// for chaining.
func (d *Drawer) Animated(on bool) *Drawer {
	if !on {
		d.duration = 0
	} else if d.duration == 0 {
		d.duration = animDuration
	}
	return d
}

// Dismissible controls whether the drawer closes itself when the scrim
// is clicked or Back (Android) or Escape is pressed. It's on by
// default; turn it off for a drawer the user must answer through its
// own controls. It returns d for chaining.
func (d *Drawer) Dismissible(on bool) *Drawer {
	d.fixed = !on
	return d
}

// SwipeToOpen lets the user open the drawer by dragging from its edge
// of the window. It's off by default: on Android with gesture
// navigation, a swipe from the screen edge is the system Back gesture,
// so an edge drawer is best opened from a button there. It returns d
// for chaining.
func (d *Drawer) SwipeToOpen(on bool) *Drawer {
	d.edgeSwipe = on
	return d
}

// Styles replaces the panel styles, from the next frame. It returns d
// for chaining.
func (d *Drawer) Styles(st styles.Styles) *Drawer {
	d.st = st
	return d
}
