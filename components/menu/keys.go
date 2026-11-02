package menu

import (
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op/clip"

	"github.com/mehran9675/giode/components/kit"
)

// navKeys are the keys an open menu takes: Up and Down move the
// highlight (scrolling it into view), Home and End jump, Right or Enter
// opens a submenu, Left goes back out of one, Enter and Space choose.
// Escape is handled with the other dismissals (see Layout).
var navKeys = []key.Name{
	key.NameUpArrow, key.NameDownArrow, key.NameHome, key.NameEnd,
	key.NameLeftArrow, key.NameRightArrow, key.NameReturn, key.NameEnter, key.NameSpace,
}

// handleKeys gives a root menu the keyboard while it's open and acts on
// the keys for the innermost menu the keyboard is in.
func (m *Menu) handleKeys(gtx layout.Context) {
	if !m.hasFocus {
		gtx.Execute(key.FocusCmd{Tag: &m.keyTag})
		m.hasFocus = true
	}
	// An empty area: the tag only needs to exist to take focus, and a
	// window-wide one would take presses from what's under the menu.
	area := clip.Rect{}.Push(gtx.Ops)
	event.Op(gtx.Ops, &m.keyTag)
	area.Pop()
	filters := make([]event.Filter, 0, len(navKeys)+1)
	filters = append(filters, key.FocusFilter{Target: &m.keyTag})
	for _, n := range navKeys {
		filters = append(filters, key.Filter{Focus: &m.keyTag, Name: n})
	}
	for {
		ev, ok := gtx.Event(filters...)
		if !ok {
			break
		}
		e, ok := ev.(key.Event)
		if !ok || e.State != key.Press {
			continue
		}
		if m.keyTarget().onKey(e.Name, kit.RTL(gtx)) {
			kit.RequestFrame(gtx)
		}
	}
}

// keyTarget is the innermost menu the keyboard has moved into.
func (m *Menu) keyTarget() *Menu {
	t := m
	for t.openSubmenu >= 0 && t.openSubmenu < len(t.items) {
		sub := t.items[t.openSubmenu].submenu
		if sub == nil || !sub.keyboardIn {
			break
		}
		t = sub
	}
	return t
}

// onKey acts on a key for this menu, reporting whether it did anything.
// Right to left, the arrows that enter and leave submenus swap.
func (m *Menu) onKey(name key.Name, rtl bool) bool {
	enter, leave := key.NameRightArrow, key.NameLeftArrow
	if rtl {
		enter, leave = leave, enter
	}
	switch name {
	case key.NameDownArrow:
		return m.moveActive(1)
	case key.NameUpArrow:
		return m.moveActive(-1)
	case key.NameHome:
		m.active = -1
		return m.moveActive(1)
	case key.NameEnd:
		m.active = len(m.items)
		return m.moveActive(-1)
	case enter:
		return m.openActiveSubmenu()
	case leave:
		if m.parent == nil {
			return false
		}
		m.keyboardIn = false
		m.parent.openSubmenu = -1
		m.Close()
		return true
	case key.NameReturn, key.NameEnter, key.NameSpace:
		if m.active < 0 || m.active >= len(m.items) {
			return false
		}
		it := &m.items[m.active]
		if it.submenu != nil {
			return m.openActiveSubmenu()
		}
		if it.action != nil {
			it.action()
		}
		m.root().Close()
		return true
	}
	return false
}

// moveActive moves the highlight by dir (±1) to the next item that can
// be chosen, wrapping around, and asks for it to be scrolled into view.
func (m *Menu) moveActive(dir int) bool {
	n := len(m.items)
	if n == 0 {
		return false
	}
	i := m.active
	for range n {
		i += dir
		switch {
		case i < 0:
			i = n - 1
		case i >= n:
			i = 0
		}
		if it := m.items[i]; !it.separator && !it.disabled {
			m.active = i
			m.root().keyNav = true
			m.scroll.reveal = true
			// A submenu the pointer opened closes as the keyboard moves on.
			if m.openSubmenu >= 0 && m.openSubmenu != i {
				if sub := m.items[m.openSubmenu].submenu; sub != nil {
					sub.Close()
				}
				m.openSubmenu = -1
			}
			return true
		}
	}
	return false
}

// openActiveSubmenu opens the highlighted item's submenu and moves the
// keyboard into it, on its first item.
func (m *Menu) openActiveSubmenu() bool {
	if m.active < 0 || m.active >= len(m.items) || m.items[m.active].submenu == nil {
		return false
	}
	sub := m.items[m.active].submenu
	m.openSubmenu = m.active
	sub.keyboardIn = true
	sub.active = -1
	sub.moveActive(1)
	return true
}
