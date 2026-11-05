// Package swipeable provides list rows that reveal actions when swiped
// sideways, like the delete and pin actions of mobile mail and chat
// apps.

package swipeable

import (
	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/widget"
	"github.com/xmhrnx/giode/components/kit"
)

// Layout lays out the row and its revealed actions.
// update handles drags and taps.
func (s *Swipeable) update(gtx layout.Context, leadW, trailW float32) {
	// Drags move the row; releasing settles it open or closed.
	for {
		e, ok := s.drag.Update(gtx.Metric, gtx.Source, gesture.Horizontal)
		if !ok {
			break
		}
		switch e.Kind {
		case pointer.Press:
			s.dragFrom, s.dragStart = e.Position.X, s.offset
		case pointer.Drag:
			dx := e.Position.X - s.dragFrom
			if !s.dragging && abs(dx) < float32(gtx.Dp(8)) {
				continue
			}
			s.dragging = true
			s.offset = max(-trailW, min(leadW, s.dragStart+dx))
			s.target = s.offset
		case pointer.Release, pointer.Cancel:
			if !s.dragging {
				continue
			}
			s.dragging = false
			switch {
			case s.offset > leadW/2 && leadW > 0:
				s.target = leadW
			case s.offset < -trailW/2 && trailW > 0:
				s.target = -trailW
			default:
				s.target = 0
			}
		}
		kit.RequestFrame(gtx)
	}

	// Taps on actions, or on the open row to close it.
	for i := range s.leading {
		if s.leadClicks[i].Clicked(gtx) {
			s.tap(s.leading[i])
		}
	}
	for i := range s.trailing {
		if s.trailClicks[i].Clicked(gtx) {
			s.tap(s.trailing[i])
		}
	}
	if s.closeClick.Clicked(gtx) {
		s.target = 0
	}

}

func (s *Swipeable) tap(a Action) {
	if a.OnTap != nil {
		a.OnTap()
	}
	s.target = 0
}

// sides returns the actions revealed on the left (by swiping right)
// and on the right: leading and trailing, swapped right to left, where
// lines start at the right edge.
func (s *Swipeable) sides(gtx layout.Context) (left []Action, leftClicks []widget.Clickable, right []Action, rightClicks []widget.Clickable) {
	if kit.RTL(gtx) {
		return s.trailing, s.trailClicks, s.leading, s.leadClicks
	}
	return s.leading, s.leadClicks, s.trailing, s.trailClicks
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
