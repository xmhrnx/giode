package text

import (
	"time"

	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"

	"unicode/utf8"

	"github.com/xmhrnx/giode/components/kit"
)

// update processes the area's events and reports whether the selection
// changed since the area was last laid out.
func (s *selArea) update(gtx layout.Context) bool {
	for {
		ev, ok := gtx.Event(
			pointer.Filter{Target: &s.tag, Kinds: pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel},
			key.FocusFilter{Target: &s.tag},
			key.Filter{Focus: &s.tag, Name: "C", Required: key.ModShortcut},
			key.Filter{Focus: &s.tag, Name: "A", Required: key.ModShortcut},
			key.Filter{Focus: &s.tag, Name: key.NameEscape},
		)
		if !ok {
			break
		}
		switch e := ev.(type) {
		case pointer.Event:
			s.pointer(gtx, e)
		case key.Event:
			if e.State != key.Press {
				break
			}
			switch e.Name {
			case "C":
				if t := s.selectedText(); t != "" {
					kit.WriteClipboard(gtx, t)
				}
			case "A":
				if n := len(s.prev); n > 0 {
					s.set(selPos{}, selPos{n - 1, utf8.RuneCountInString(s.prev[n-1].text)})
				}
			case key.NameEscape:
				s.clear()
			}
		}
	}
	if s.pressing && kit.Scrolling() {
		// A finger resting during a scroll isn't a long-press.
		s.pressing = false
	}
	if s.pressing {
		if gtx.Now.Sub(s.pressAt) >= longPress {
			s.pressing = false
			if pos, ok := s.hit(s.abs(s.pressPos)); ok {
				s.set(s.word(pos))
				s.touchSel = true
				// Keep the finger from scrolling what's underneath.
				gtx.Execute(pointer.GrabCmd{Tag: &s.tag, ID: s.pressID})
			}
		} else {
			kit.RequestFrameAt(gtx, s.pressAt.Add(longPress))
		}
	}
	return s.version != s.seen
}

func (s *selArea) pointer(gtx layout.Context, e pointer.Event) {
	if e.Kind == pointer.Cancel {
		// Another handler grabbed the pointer - a list scrolling. Gio's
		// cancel event carries no source, so it ends any gesture.
		s.pressing, s.dragging = false, false
		if s.touchSel {
			s.touchSel = false
			s.showCopy = !s.empty()
		}
		return
	}
	if e.Source == pointer.Touch {
		s.touch(gtx, e)
		return
	}
	switch e.Kind {
	case pointer.Press:
		if e.Buttons.Contain(pointer.ButtonSecondary) {
			s.showCopy = !s.empty()
			return
		}
		if !e.Buttons.Contain(pointer.ButtonPrimary) {
			return
		}
		s.showCopy = false
		pos, ok := s.hit(s.abs(e.Position))
		if !ok {
			s.clear()
			return
		}
		gtx.Execute(key.FocusCmd{Tag: &s.tag})
		d := e.Position.Sub(s.lastPos)
		if gtx.Now.Sub(s.lastClick) < 400*time.Millisecond && d.X*d.X+d.Y*d.Y < 25 {
			s.clicks++
		} else {
			s.clicks = 1
		}
		s.lastClick, s.lastPos = gtx.Now, e.Position
		switch {
		case e.Modifiers.Contain(key.ModShift) && s.has:
			s.set(s.anchor, pos)
		case s.clicks == 2:
			s.set(s.word(pos))
		case s.clicks >= 3:
			s.set(selPos{pos.part, 0}, selPos{pos.part, utf8.RuneCountInString(s.prev[pos.part].text)})
		default:
			s.set(pos, pos)
		}
		s.dragging = true
	case pointer.Drag:
		if !s.dragging {
			return
		}
		if pos, ok := s.hit(s.abs(e.Position)); ok {
			s.set(s.anchor, pos)
		}
	case pointer.Release, pointer.Cancel:
		s.dragging = false
	}
}

func (s *selArea) touch(gtx layout.Context, e pointer.Event) {
	switch e.Kind {
	case pointer.Press:
		s.pressing, s.pressAt, s.pressPos, s.pressID = true, gtx.Now, e.Position, e.PointerID
		kit.RequestFrameAt(gtx, gtx.Now.Add(longPress))
	case pointer.Drag:
		if s.touchSel && e.PointerID == s.pressID {
			if pos, ok := s.hit(s.abs(e.Position)); ok {
				s.set(s.anchor, pos)
			}
			return
		}
		d := e.Position.Sub(s.pressPos)
		if slop := float32(gtx.Dp(8)); d.X*d.X+d.Y*d.Y > slop*slop {
			s.pressing = false // a scroll, not a long-press
		}
	case pointer.Release, pointer.Cancel:
		s.pressing = false
		if s.touchSel {
			s.touchSel = false
			s.showCopy = e.Kind == pointer.Release && !s.empty()
		} else if e.Kind == pointer.Release {
			s.clear()
		}
	}
}

func (s *selArea) passFunc() func(gtx layout.Context) bool {
	if s.pass == nil {
		s.pass = s.update
	}
	return s.pass
}
