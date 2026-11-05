// Package menu provides a right-click context menu that opens at the
// cursor and keeps itself within the bounds of the window, flipping
// above the cursor when it would overflow the bottom edge.

package menu

import (
	"gioui.org/f32"
)

// OpenAt opens the menu anchored at p, in window pixel coordinates.
func (m *Menu) OpenAt(p f32.Point) {
	m.pos = p
	m.open = true
}

// Toggle flips the menu state, anchored at p.
func (m *Menu) Toggle(p f32.Point) {
	if m.open {
		m.Close()
		return
	}
	m.OpenAt(p)
}

// Close closes the menu, and any open submenu (recursively, in case a
// submenu itself has a submenu open).
func (m *Menu) Close() {
	m.open = false
	m.openSubmenu = -1
	m.active, m.keyNav, m.keyboardIn, m.hasFocus = -1, false, false, false
	m.scroll.offset = 0
	for i := range m.items {
		if sub := m.items[i].submenu; sub != nil {
			sub.Close()
		}
	}
}

// Opened reports whether the menu is open.
func (m *Menu) Opened() bool {
	return m.open
}
