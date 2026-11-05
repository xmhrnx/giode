package box

import (
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"

	"github.com/xmhrnx/giode/components/kit"
)

func (p *Pressable) fire(e ClickEvent) {
	p.fired++
	if p.onClick != nil {
		p.onClick(e)
	}
	if e.Clicks == 2 && e.Button == MousePrimary && p.onDouble != nil {
		p.onDouble(e)
	}
	if e.Button == MouseSecondary && p.onSecondary != nil {
		p.onSecondary(e)
	}
}

// update processes the events queued since the last frame.
func (p *Pressable) update(gtx layout.Context) (changed bool) {
	if p.disabled {
		return false
	}
	for {
		ev, ok := gtx.Event(pointer.Filter{
			Target: p,
			Kinds:  pointer.Press | pointer.Release | pointer.Enter | pointer.Leave | pointer.Cancel,
		})
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		changed = true
		switch e.Kind {
		case pointer.Enter:
			p.hovered = true
		case pointer.Leave:
			p.hovered = false
		case pointer.Cancel:
			p.hovered, p.pressed = false, false
		case pointer.Press:
			if p.pressed {
				break
			}
			p.pressed = true
			p.pressPos = e.Position
			p.pressSrc = e.Source
			p.pressAt = gtx.Now
			p.longFired = false
			if e.Source == pointer.Touch && p.onLongPress != nil {
				kit.RequestFrameAt(gtx, gtx.Now.Add(LongPressDuration))
			}
			switch {
			case e.Buttons.Contain(pointer.ButtonSecondary):
				p.pressBtn = MouseSecondary
			case e.Buttons.Contain(pointer.ButtonTertiary):
				p.pressBtn = MouseMiddle
			default:
				p.pressBtn = MousePrimary
			}
			// A selectable box leaves the keyboard focus to the
			// selection, so Ctrl+C copies what the press selects.
			if p.focusable && e.Source == pointer.Mouse && !p.selectable {
				gtx.Execute(key.FocusCmd{Tag: p})
			}
		case pointer.Release:
			if !p.pressed {
				break
			}
			p.pressed = false
			// Released outside: not a click. Gio keeps a pressed
			// touch's Enter/Leave up to date as the finger slides, and
			// delivers the release before the touch's final Leave, so
			// this holds for touch as well as the mouse.
			if !p.hovered || p.longFired {
				break
			}
			if p.selectable && p.pressSrc == pointer.Mouse {
				// A mouse drag across selectable text selects it.
				slop := float32(gtx.Dp(multiClickSlop))
				if d := e.Position.Sub(p.pressPos); d.X*d.X+d.Y*d.Y > slop*slop {
					break
				}
			}
			slop := float32(gtx.Dp(multiClickSlop))
			d := e.Position.Sub(p.lastClickPos)
			if gtx.Now.Sub(p.lastClick) <= multiClickWindow && d.X*d.X+d.Y*d.Y <= slop*slop {
				p.clicks++
			} else {
				p.clicks = 1
			}
			p.lastClick, p.lastClickPos = gtx.Now, e.Position
			p.fire(ClickEvent{
				Clicks: p.clicks, Button: p.pressBtn, Modifiers: e.Modifiers,
				Source: ClickPointer, Position: e.Position.Round(), PositionDp: toDp(e.Position, gtx),
				WindowPosition: p.origin.Add(e.Position.Round()),
			})
		}
	}
	if p.pressed && p.pressSrc == pointer.Touch && p.onLongPress != nil && !p.longFired {
		switch {
		case !p.hovered:
			// Slid off: neither a long-press nor a click.
		case gtx.Now.Sub(p.pressAt) >= LongPressDuration:
			p.longFired = true
			changed = true
			if fn := longPressFeedback.Load(); fn != nil {
				(*fn)()
			}
			p.fired++
			p.onLongPress(ClickEvent{Clicks: 1, Source: ClickPointer, Position: p.pressPos.Round(), PositionDp: toDp(p.pressPos, gtx), WindowPosition: p.origin.Add(p.pressPos.Round())})
		default:
			kit.RequestFrameAt(gtx, p.pressAt.Add(LongPressDuration))
		}
	}
	if p.focusable {
		for {
			ev, ok := gtx.Event(
				key.FocusFilter{Target: p},
				key.Filter{Focus: p, Name: key.NameReturn},
				key.Filter{Focus: p, Name: key.NameEnter},
				key.Filter{Focus: p, Name: key.NameSpace},
			)
			if !ok {
				break
			}
			changed = true
			switch e := ev.(type) {
			case key.FocusEvent:
				p.focused = e.Focus
				p.pressedKey = ""
			case key.Event:
				if !gtx.Focused(p) {
					break
				}
				switch e.State {
				case key.Press:
					p.pressedKey = e.Name
				case key.Release:
					if p.pressedKey == e.Name {
						p.pressedKey = ""
						p.fire(ClickEvent{Clicks: 1, Modifiers: e.Modifiers, Source: ClickKeyboard})
					}
				}
			}
		}
	} else {
		p.focused = false
	}
	if changed {
		kit.RequestFrame(gtx)
	}
	return changed
}

// pass is the Pressable's input pass (see kit.HandleInput): it handles
// clicks, hover and focus while an enclosing cache replays it, and
// reports a change in the state its content is built from, or a fired
// handler.
func (p *Pressable) pass(gtx layout.Context) bool {
	before, fired := p.State(), p.fired
	if !p.update(gtx) {
		return false
	}
	return p.State() != before || p.fired != fired
}
