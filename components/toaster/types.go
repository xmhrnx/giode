// Package toaster shows small notifications that pop up in a corner of
// the window and disappear on their own.

package toaster

import (
	"time"

	"github.com/xmhrnx/giode/styles/properties"
)

// Kind selects a toast's accent color.
type Kind int

const (
	Info Kind = iota
	Success
	Warning
	Error
)

func (k Kind) color() properties.Color {
	switch k {
	case Success:
		return "theme.success"
	case Warning:
		return "theme.warning"
	case Error:
		return "theme.danger"
	default:
		return "theme.info"
	}
}

// Placement selects the window corner or edge toasts appear at.
type Placement int

const (
	BottomRight Placement = iota
	BottomLeft
	BottomCenter
	TopRight
	TopLeft
	TopCenter
	// BottomEnd, BottomStart, TopEnd and TopStart follow the reading
	// direction: BottomEnd is the bottom right, or the bottom left in a
	// right-to-left locale. BottomEnd is the default.
	BottomEnd
	BottomStart
	TopEnd
	TopStart
)

func (p Placement) top() bool {
	switch p {
	case TopRight, TopLeft, TopCenter, TopEnd, TopStart:
		return true
	}
	return false
}

// physical resolves the start and end placements for a layout reading
// right to left or not.
func (p Placement) physical(rtl bool) Placement {
	switch p {
	case BottomEnd, BottomStart, TopEnd, TopStart:
		end := p == BottomEnd || p == TopEnd
		right := end != rtl
		switch {
		case p.top() && right:
			return TopRight
		case p.top():
			return TopLeft
		case right:
			return BottomRight
		}
		return BottomLeft
	}
	return p
}

// DefaultDuration is how long a toast stays up, not counting time
// spent hovered.
const DefaultDuration = 5 * time.Second

const (
	toastWidth = 340 // dp
	margin     = 16  // dp, from the window edges
	gap        = 8   // dp, between toasts
)
