// Package swipeable provides list rows that reveal actions when swiped
// sideways, like the delete and pin actions of mobile mail and chat
// apps.

package swipeable

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/unit"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
)

func (s *Swipeable) Layout(gtx layout.Context) layout.Dimensions {
	dims := s.layoutSelf(gtx)
	// A right-click or long-press anywhere on it opens its context menu.
	s.AnchorContext(gtx, dims.Size)
	return dims
}

// layoutSelf lays it out without its context menu area.
func (s *Swipeable) layoutSelf(gtx layout.Context) layout.Dimensions {
	aw := gtx.Dp(unit.Dp(actionWidth))
	left, leftClicks, right, rightClicks := s.sides(gtx)
	leftW, rightW := float32(aw*len(left)), float32(aw*len(right))
	// SetActions may have changed the widths under an open row.
	switch {
	case s.target > 0:
		s.target = leftW
	case s.target < 0:
		s.target = -rightW
	}

	s.update(gtx, leftW, rightW)
	if s.pass == nil {
		s.pass = func(gtx layout.Context) bool {
			aw := gtx.Dp(unit.Dp(actionWidth))
			left, _, right, _ := s.sides(gtx)
			off, target := s.offset, s.target
			s.update(gtx, float32(aw*len(left)), float32(aw*len(right)))
			return s.offset != off || s.target != target
		}
	}
	kit.HandleInput(s.pass)

	// Animate toward the target.
	dt := gtx.Now.Sub(s.last)
	if s.last.IsZero() || dt < 0 || dt > animDuration {
		dt = animDuration / 10
	}
	s.last = gtx.Now
	if !s.dragging && s.offset != s.target {
		step := float32(aw*max(len(s.leading), len(s.trailing), 1)) * float32(dt) / float32(animDuration)
		if s.offset < s.target {
			s.offset = min(s.target, s.offset+step)
		} else {
			s.offset = max(s.target, s.offset-step)
		}
	}
	if s.offset != s.target || s.dragging {
		kit.RequestFrame(gtx)
	}
	if s.offset != 0 {
		kit.MarkLive()
	}

	// The content, recorded so the actions can be sized to its height.
	off := image.Pt(int(s.offset), 0)
	track := elements.TrackChild()
	macro := op.Record(gtx.Ops)
	dims := s.content().Layout(gtx)
	call := macro.Stop()
	track.End()
	size := dims.Size

	// The drag area is the row's root, so the content's own clicks keep
	// working inside it.
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	s.drag.Add(gtx.Ops)

	rtl := kit.RTL(gtx)
	if s.offset > 0 {
		s.layoutActions(gtx, left, leftClicks, image.Rect(0, 0, int(s.offset), size.Y), 0, aw, rtl)
	}
	if s.offset < 0 {
		s.layoutActions(gtx, right, rightClicks, image.Rect(size.X+int(s.offset), 0, size.X, size.Y), size.X-int(rightW), aw, rtl)
	}

	track.Place(off)
	func() {
		defer op.Offset(off).Push(gtx.Ops).Pop()
		call.Add(gtx.Ops)
		if s.target != 0 && !s.dragging {
			// While open, a tap on the row closes it instead of
			// reaching the content.
			s.closeClick.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Dimensions{Size: size}
			})
		}
	}()
	return dims
}
