// Package scroll provides a stateful vertical scroll container.

package scroll

import (
	"gioui.org/gesture"
	"gioui.org/layout"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
)

// Scroll is a stateful vertical scroll container. It implements
// elements.Element, so it can be placed directly in the tree like any
// other element. Create it once with New and lay it out every frame;
// content is rebuilt every frame while visible and may be taller than
// the viewport.
//
// The scroll offset reacts to the mouse wheel and touch drags, and
// the scrollbar thumb can be dragged with the mouse.
type Scroll struct {
	// Contextual gives it a context menu (Context).
	elements.Contextual

	offset        float32
	scroll        gesture.Scroll
	bar           kit.Scrollbar
	contentH      int
	viewportH     int
	content       func() elements.Element
	st            styles.Styles
	onReachBottom func()
	wasAtBottom   bool
	pxPerDp       float32

	axis Axis
	// Horizontal scrolling: offset, sizes and scrollbar.
	offsetX float32
	// rtl records that the last layout read right to left, where
	// offsetX counts from the right edge.
	rtl       bool
	scrollX   gesture.Scroll
	gest      kit.ScrollGesture
	gestX     kit.ScrollGesture
	contentW  int
	viewportW int
	barX      kit.Scrollbar

	anchor Anchor
	// following is set while an AnchorBottom scroll keeps to the
	// bottom: until the user scrolls away from it.
	following bool
	fit       bool
	onScroll  func(offset, max float32)
	// movedBy is set by ScrollBy until the next layout settles it.
	movedBy bool
	// pass is the input pass (see kit.HandleInput).
	pass     func(gtx layout.Context) bool
	lastOff  [2]float32
	measured bool
}

// New returns a Scroll around content, rebuilt fresh every frame. The
// styles argument is optional: Background is the container fill and
// ScrollBar customizes the scrollbar's width, colors, corner radius
// and minimum thumb length.
func New(content func() elements.Element, st ...styles.Styles) *Scroll {
	s := &Scroll{content: content}
	if len(st) > 0 {
		s.st = st[0]
	}
	return s
}

// Axis sets the directions the content scrolls in: Vertical (the
// default), Horizontal, or Vertical|Horizontal. Along a scrolling
// axis the content gets unlimited room; along the other it gets the
// viewport's size. It returns s for chaining.
func (s *Scroll) Axis(a Axis) *Scroll {
	s.axis = a
	return s
}

// Anchor sets where the content stays as it grows; see AnchorBottom.
// It returns s for chaining.
func (s *Scroll) Anchor(a Anchor) *Scroll {
	s.anchor = a
	s.following = a == AnchorBottom
	return s
}

// FitContent makes the viewport only as tall (or, scrolling
// horizontally, as wide) as the content, up to the available space,
// instead of always filling it: a sheet with three rows stays three
// rows tall and scrolls once the content outgrows the screen. It
// returns s for chaining.
func (s *Scroll) FitContent(on bool) *Scroll {
	s.fit = on
	return s
}

// OnScroll sets a handler called after the offset changes - by the
// user, ScrollTo or an anchor - with the vertical offset and its
// maximum in pixels (the horizontal ones for a Horizontal-only
// scroll). It returns s for chaining.
func (s *Scroll) OnScroll(fn func(offset, max float32)) *Scroll {
	s.onScroll = fn
	return s
}

// Styles replaces the styles, from the next frame. It returns s for
// chaining.
func (s *Scroll) Styles(st styles.Styles) *Scroll {
	s.st = st
	return s
}

// OffsetX returns the horizontal offset in pixels, from the start
// edge: the left, or the right in a right-to-left locale, where a
// horizontal Scroll starts at its right end as in a browser.
func (s *Scroll) OffsetX() float32 {
	return s.offsetX
}

// ScrollToX sets the horizontal offset in pixels, from the start edge
// (see OffsetX).
func (s *Scroll) ScrollToX(v float32) {
	s.offsetX = v
}

// ScrollToBottom scrolls to the end of the content, and resumes
// following it with AnchorBottom.
func (s *Scroll) ScrollToBottom() {
	s.offset = unbounded
	if s.anchor == AnchorBottom {
		s.following = true
	}
}

// ScrollTo sets the vertical offset in pixels.
func (s *Scroll) ScrollTo(v float32) {
	s.offset = v
}

// Offset returns the current vertical offset.
func (s *Scroll) Offset() float32 {
	return s.offset
}

// OffsetDp returns the vertical offset in dp.
func (s *Scroll) OffsetDp() float32 {
	scale := s.pxPerDp
	if scale <= 0 {
		scale = 1
	}
	return s.offset / scale
}

// ScrollToDp sets the vertical offset in dp.
func (s *Scroll) ScrollToDp(v float32) {
	scale := s.pxPerDp
	if scale <= 0 {
		scale = elements.DisplayScale()
	}
	s.offset = v * scale
}

// ScrollBy moves the content by dx, dy pixels, as a wheel or drag
// would: positive dy scrolls down. The offsets are clamped to the
// content on the next layout. It's how automation (the MCP server's
// ui_scroll) scrolls.
func (s *Scroll) ScrollBy(dx, dy float32) {
	s.offset += dy
	if s.rtl {
		// The offset counts from the right edge.
		dx = -dx
	}
	s.offsetX += dx
	if dy < 0 {
		s.following = false
	}
	s.movedBy = true
}

// ScrollOffset returns the horizontal and vertical offsets in pixels.
func (s *Scroll) ScrollOffset() (x, y float32) {
	return s.offsetX, s.offset
}

// notify calls OnScroll when an offset changed since the last call.
func (s *Scroll) notify() {
	cur := [2]float32{s.offset, s.offsetX}
	if cur == s.lastOff {
		return
	}
	s.lastOff = cur
	if s.onScroll == nil {
		return
	}
	if s.vertical() {
		s.onScroll(s.offset, float32(s.maxOffset()))
	} else {
		s.onScroll(s.offsetX, float32(s.maxOffsetX()))
	}
}
