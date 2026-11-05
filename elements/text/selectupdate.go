package text

import (
	"gioui.org/io/pointer"
	"gioui.org/layout"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements/internal/base"
)

// update processes presses on the text: a touch held still for
// longPress, or a right-click, selects it all.
func (s *selectState) update(gtx layout.Context) {
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &s.tap, Kinds: pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel})
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		switch e.Kind {
		case pointer.Press:
			switch {
			case e.Source == pointer.Touch:
				s.pressing, s.pressAt, s.pressPos = true, gtx.Now, e.Position
				kit.RequestFrameAt(gtx, gtx.Now.Add(longPress))
			case e.Buttons.Contain(pointer.ButtonSecondary):
				s.selectAll()
				kit.RequestFrame(gtx)
			}
		case pointer.Drag:
			d := e.Position.Sub(s.pressPos)
			if slop := float32(gtx.Dp(8)); d.X*d.X+d.Y*d.Y > slop*slop {
				s.pressing = false // a scroll or a drag, not a long-press
			}
		case pointer.Release, pointer.Cancel:
			s.pressing = false
		}
	}
	if s.pressing {
		if gtx.Now.Sub(s.pressAt) >= longPress {
			s.pressing = false
			s.selectAll()
		} else {
			kit.RequestFrameAt(gtx, s.pressAt.Add(longPress))
		}
	}
	if s.all {
		base.HandleDismiss(func() { s.all = false })
		kit.MarkLive()
	}
}

// passFunc returns the input pass (see kit.HandleInput): long-presses
// and right-clicks select it all.
func (s *selectState) passFunc() func(gtx layout.Context) bool {
	if s.pass == nil {
		s.pass = func(gtx layout.Context) bool {
			was := s.all
			s.update(gtx)
			return s.all != was
		}
	}
	return s.pass
}
