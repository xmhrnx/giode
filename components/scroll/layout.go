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
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles/properties"
)

// Layout lays out the scrollable content.
func (s *Scroll) Layout(gtx layout.Context) layout.Dimensions {
	dims := s.layoutSelf(gtx)
	// A right-click or long-press anywhere on it opens its context menu.
	s.AnchorContext(gtx, dims.Size)
	return dims
}

// layoutSelf lays it out without its context menu area.
func (s *Scroll) layoutSelf(gtx layout.Context) layout.Dimensions {
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

func (s *Scroll) vertical() bool { return s.axis == 0 || s.axis&Vertical != 0 }

func (s *Scroll) horizontal() bool { return s.axis&Horizontal != 0 }

// maxOffsetX returns the largest valid horizontal offset.
func (s *Scroll) maxOffsetX() int {
	return max(0, s.contentW-s.viewportW)
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
