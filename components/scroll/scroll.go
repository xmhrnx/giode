// Package scroll provides a stateful vertical scroll container.
package scroll

import (
	"image"

	"gioui.org/f32"

	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/styles/properties"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
)

const (
	// unbounded mirrors the sentinel Gio uses for infinite
	// constraints.
	unbounded = 1e6
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

// Axis selects the directions a Scroll scrolls in.
type Axis int

const (
	// Vertical scrolls up and down (the default).
	Vertical Axis = 1 << iota
	// Horizontal scrolls sideways, for wide code or tables.
	Horizontal
)

// Anchor selects where a Scroll keeps its content when it grows.
type Anchor int

const (
	// AnchorTop keeps the offset when content is added (the default).
	AnchorTop Anchor = iota
	// AnchorBottom keeps a scroll that's at the bottom there as its
	// content grows - a chat following a reply as it streams in - and
	// stops following once the user scrolls up, until they scroll
	// back to the bottom. It starts at the bottom.
	AnchorBottom
)

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

func (s *Scroll) vertical() bool   { return s.axis == 0 || s.axis&Vertical != 0 }
func (s *Scroll) horizontal() bool { return s.axis&Horizontal != 0 }

// maxOffsetX returns the largest valid horizontal offset.
func (s *Scroll) maxOffsetX() int {
	return max(0, s.contentW-s.viewportW)
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

// ScrollTo sets the vertical offset in pixels.
func (s *Scroll) ScrollTo(v float32) {
	s.offset = v
}

// Offset returns the current vertical offset.
func (s *Scroll) Offset() float32 {
	return s.offset
}

// ContentHeight returns the total height of the scrolled content, as
// measured on the last frame.
func (s *Scroll) ContentHeight() int {
	return s.contentH
}

// ViewportHeight returns the height of the visible viewport, as
// measured on the last frame.
func (s *Scroll) ViewportHeight() int {
	return s.viewportH
}

// Progress returns how far scrolled, from 0 (top) to 1 (bottom); 0
// when there's nothing to scroll (the content fits the viewport).
func (s *Scroll) Progress() float32 {
	max := s.maxOffset()
	if max <= 0 {
		return 0
	}
	return s.offset / float32(max)
}

// AtBottom reports whether the scroll is at (or past) its maximum
// offset - also true when there's nothing to scroll at all (the
// content already fits the viewport), the same way a page with no
// scrollbar is trivially "at the bottom": there's nowhere else to go,
// so a load-more handler driven by AtBottom/OnReachBottom fires
// immediately, prompting an initial fetch when the first page of
// content doesn't even fill the viewport.
func (s *Scroll) AtBottom() bool {
	return s.offset >= float32(s.maxOffset())
}

// Visible reports whether the vertical range [top, top+height) - in
// the same coordinate space as Content's own output, e.g. a running
// offset tracked while building a long list - is at least partially
// within the current viewport. Useful for lazy-loading or skipping
// expensive work for an item that isn't currently on screen.
func (s *Scroll) Visible(top, height int) bool {
	return top+height > int(s.offset) && top < int(s.offset)+s.viewportH
}

// VisibleDp is Visible with top and height in dp, the units Styles
// use, converted with the display scale of the last frame.
func (s *Scroll) VisibleDp(top, height float32) bool {
	scale := s.pxPerDp
	if scale <= 0 {
		scale = 1
	}
	return s.Visible(int(top*scale), int(height*scale+0.5))
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

// OnReachBottom sets the handler invoked once when the scroll reaches
// its maximum offset - edge-triggered: it fires again only after
// scrolling away from the bottom and back, not on every frame spent
// there. It returns s for chaining.
func (s *Scroll) OnReachBottom(fn func()) *Scroll {
	s.onReachBottom = fn
	return s
}

// maxOffset returns the largest valid offset.
func (s *Scroll) maxOffset() int {
	m := s.contentH - s.viewportH
	if m < 0 {
		return 0
	}
	return m
}

// clamp bounds v to the valid offset range.
func (s *Scroll) clamp(v float32) float32 {
	maxOff := float32(s.maxOffset())
	if v < 0 {
		return 0
	}
	if v > maxOff {
		return maxOff
	}
	return v
}

// Layout lays out the scrollable content.
func (s *Scroll) Layout(gtx layout.Context) layout.Dimensions {
	s.pxPerDp = gtx.Metric.PxPerDp
	s.rtl = kit.RTL(gtx)
	vert, horiz := s.vertical(), s.horizontal()

	// Lay the content out with unlimited room along the scrolling
	// axes.
	cgtx := gtx
	if vert {
		cgtx.Constraints.Min.Y = 0
		cgtx.Constraints.Max.Y = unbounded
	}
	if horiz {
		cgtx.Constraints.Min.X = 0
		cgtx.Constraints.Max.X = unbounded
	} else if !s.fit || vert {
		cgtx.Constraints.Min.X = gtx.Constraints.Max.X
	}
	macro := op.Record(gtx.Ops)
	track := elements.TrackChild()
	popSem := elements.PushSemanticScroll(s)
	dims := s.content().Layout(cgtx)
	popSem()
	track.End()
	contentOps := macro.Stop()

	// The viewport fills the space, or with FitContent hugs the
	// content up to it.
	viewport := gtx.Constraints.Max
	if s.fit && vert {
		viewport.Y = dims.Size.Y
	}
	if s.fit && horiz {
		viewport.X = dims.Size.X
	}
	if !vert {
		// Scrolling only sideways, it's as tall as its content, like a
		// row.
		viewport.Y = dims.Size.Y
	}
	viewport = gtx.Constraints.Constrain(viewport)
	if !s.measured {
		s.measured = true
		s.following = s.anchor == AnchorBottom
	}
	follow := s.following
	s.contentH, s.viewportH = dims.Size.Y, viewport.Y
	s.contentW, s.viewportW = dims.Size.X, viewport.X
	if !vert {
		s.contentH = viewport.Y
	}
	if !horiz {
		s.contentW = viewport.X
	}
	if follow {
		s.offset = float32(s.maxOffset())
	}

	// Wheel and touch scrolling. The scroll region covers the
	// viewport and must be registered below the content so
	// interactive children keep priority.
	defer clip.Rect{Max: viewport}.Push(gtx.Ops).Pop()
	// Keep touch-drag scrolling working even inside a frameless
	// window's whole-window drag area - see slider.Layout's identical
	// comment.
	kit.BlockDragHere(gtx)
	if vert {
		s.scroll.Add(gtx.Ops)
	}
	if horiz {
		// A gesture per axis: a drag scrolls along whichever axis it
		// moves in first, and a nested scroll on the other axis (code
		// inside a chat) passes the rest on.
		s.scrollX.Add(gtx.Ops)
	}

	// Clamp the offsets to the content.
	s.offset = s.clamp(s.offset)
	s.offsetX = max(0, min(s.offsetX, float32(s.maxOffsetX())))
	if s.movedBy {
		// ScrollBy moved it: follow the content again only at the
		// bottom, as after the user scrolls.
		s.movedBy = false
		if s.anchor == AnchorBottom {
			s.following = s.AtBottom()
		}
	}

	dist, distX := s.updateGestures(gtx)
	if dist != 0 || distX != 0 || s.scroll.State() != gesture.StateIdle || s.scrollX.State() != gesture.StateIdle {
		// Flings keep scrolling between input events; cached
		// ancestors must keep laying the content out.
		kit.MarkLive()
	}
	if s.pass == nil {
		s.pass = func(gtx layout.Context) bool {
			y, x := s.offset, s.offsetX
			s.updateGestures(gtx)
			bar := s.vertical() && s.bar.Update(gtx, &s.offset)
			if bar && s.anchor == AnchorBottom {
				s.following = s.AtBottom()
			}
			return bar || y != s.offset || x != s.offsetX
		}
	}
	kit.HandleInput(s.pass)

	atBottom := s.AtBottom()
	if atBottom && !s.wasAtBottom && s.onReachBottom != nil {
		s.onReachBottom()
	}
	s.wasAtBottom = atBottom

	// Draw the scrolled content.
	if properties.CalcColor(s.st.Background).A != 0 {
		paint.Fill(gtx.Ops, properties.CalcColor(s.st.Background))
	}
	// The content was laid out before the offset was final: move what
	// it registered (dropdown anchors, semantic nodes) with it.
	off := image.Pt(-int(s.viewLeft()), -int(s.offset))
	track.Place(off)
	offOp := op.Offset(off).Push(gtx.Ops)
	contentOps.Add(gtx.Ops)
	offOp.Pop()

	if vert {
		s.layoutScrollbar(gtx, viewport)
	}
	if horiz {
		s.layoutScrollbarX(gtx, viewport)
	}
	if s.anchor == AnchorBottom && s.bar.Dragging() {
		s.following = s.AtBottom()
	}
	s.notify()
	elements.RecordSemantic(gtx, elements.SemanticNode{Kind: elements.SemanticScroll, Target: s}, viewport)
	return layout.Dimensions{Size: viewport}
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

// updateGestures applies wheel and touch scrolling to the offsets,
// within the content measured by the last layout, and returns the
// distances.
func (s *Scroll) updateGestures(gtx layout.Context) (dist, distX int) {
	if s.vertical() {
		rangeY := pointer.ScrollRange{Min: -int(s.offset), Max: s.maxOffset() - int(s.offset)}
		dist = s.scroll.Update(gtx.Metric, gtx.Source, gtx.Now, gesture.Vertical, pointer.ScrollRange{}, rangeY)
		if dist != 0 {
			s.offset = s.clamp(s.offset + float32(dist))
		}
		started, ended := s.gest.Track(&s.scroll, dist)
		if s.anchor == AnchorBottom {
			switch {
			case started || s.gest.Moving():
				// A finger is moving the content: don't pull it back
				// to the bottom as the content grows.
				s.following = false
			case ended:
				s.following = s.AtBottom()
			}
		}
	}
	if s.horizontal() {
		// The gesture works in physical pixels from the left; right to
		// left, offsetX counts from the right.
		left := s.viewLeft()
		rangeX := pointer.ScrollRange{Min: -int(left), Max: s.maxOffsetX() - int(left)}
		distX = s.scrollX.Update(gtx.Metric, gtx.Source, gtx.Now, gesture.Horizontal, rangeX, pointer.ScrollRange{})
		if distX != 0 {
			d := float32(distX)
			if s.rtl {
				d = -d
			}
			s.offsetX = max(0, min(s.offsetX+d, float32(s.maxOffsetX())))
		}
		s.gestX.Track(&s.scrollX, distX)
	}
	if s.anchor == AnchorBottom && !s.gest.Moving() && (dist != 0 || s.bar.Dragging()) {
		// The user scrolled: follow the content again only once they're
		// back at the bottom.
		s.following = s.AtBottom()
	}
	return dist, distX
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

// layoutScrollbarX draws the horizontal scrollbar along the bottom
// edge: the vertical one, with the axes swapped.
func (s *Scroll) layoutScrollbarX(gtx layout.Context, viewport image.Point) {
	// Right to left, the thumb measures from the right edge, as the
	// offset does.
	defer kit.MirrorRTL(gtx, viewport.X)()
	swap := f32.NewAffine2D(0, 1, 0, 1, 0, 0)
	defer op.Affine(swap).Push(gtx.Ops).Pop()
	sgtx := gtx
	sgtx.Constraints = layout.Exact(image.Pt(viewport.Y, viewport.X))
	s.barX.Layout(sgtx, image.Pt(viewport.Y, viewport.X), s.contentW, &s.offsetX, s.st.ScrollBar)
}

// layoutScrollbar draws the track and draggable thumb when the
// content overflows, fully customized via st.ScrollBar: along the right
// edge, or the left one right to left.
func (s *Scroll) layoutScrollbar(gtx layout.Context, viewport image.Point) {
	// Right to left, along the left edge, as browsers place it.
	defer kit.MirrorRTL(gtx, viewport.X)()
	s.bar.Layout(gtx, viewport, s.contentH, &s.offset, s.st.ScrollBar)
}

// viewLeft is the visible area's left edge in the content, in pixels.
func (s *Scroll) viewLeft() float32 {
	if s.rtl {
		return float32(s.maxOffsetX()) - s.offsetX
	}
	return s.offsetX
}
