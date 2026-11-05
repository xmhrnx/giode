// Package swipeable provides list rows that reveal actions when swiped
// sideways, like the delete and pin actions of mobile mail and chat
// apps.

package swipeable

import (
	"time"

	"gioui.org/gesture"
	"gioui.org/layout"
	"gioui.org/widget"
	"github.com/xmhrnx/giode/elements"
)

// Swipeable is a row that slides sideways to reveal actions: leading
// ones on the left when swiped right, trailing ones on the right when
// swiped left (the other way round in a right-to-left locale). A swipe past half the actions' width opens the row; a
// tap on the row or an action closes it. Vertical drags still scroll
// the list around it.
//
// Create it once per row (in a dynamic list, with Keyed) and lay it out
// every frame. The content should paint an opaque background, since
// the actions sit behind it.
type Swipeable struct {
	content           func() elements.Element
	leading, trailing []Action
	leadClicks        []widget.Clickable
	trailClicks       []widget.Clickable
	closeClick        widget.Clickable

	// offset is how far the content is moved right (positive) or left,
	// in pixels; target is where it settles.
	offset, target float32
	last           time.Time
	drag           gesture.Drag
	dragFrom       float32
	dragStart      float32
	dragging       bool
	pass           func(gtx layout.Context) bool
}

// New returns a row showing content, with leading and trailing actions
// (either may be nil).
func New(content func() elements.Element, leading, trailing []Action) *Swipeable {
	return &Swipeable{
		content: content, leading: leading, trailing: trailing,
		leadClicks: make([]widget.Clickable, len(leading)), trailClicks: make([]widget.Clickable, len(trailing)),
	}
}

// SetActions replaces the row's leading and trailing actions (either
// may be nil), for labels that follow state, a Pin that becomes Unpin,
// say. Call it before Layout whenever they change; an open row stays
// open as long as its side still has actions, resized to fit them.
func (s *Swipeable) SetActions(leading, trailing []Action) {
	s.leading, s.trailing = leading, trailing
	s.leadClicks = resize(s.leadClicks, len(leading))
	s.trailClicks = resize(s.trailClicks, len(trailing))
}

// resize returns clicks with n entries, keeping the existing ones so a
// press in progress survives a relabel.
func resize(clicks []widget.Clickable, n int) []widget.Clickable {
	if n <= len(clicks) {
		return clicks[:n]
	}
	return append(clicks, make([]widget.Clickable, n-len(clicks))...)
}

// Close slides the row back.
func (s *Swipeable) Close() {
	s.target = 0
}

// Opened reports whether the row is open, or opening.
func (s *Swipeable) Opened() bool {
	return s.target != 0
}
