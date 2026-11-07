// Package menu provides a right-click context menu that opens at the
// cursor and keeps itself within the bounds of the window, flipping
// above the cursor when it would overflow the bottom edge.

package menu

import (
	"image"

	"gioui.org/f32"

	"github.com/xmhrnx/giode/elements"
)

// OpenAt opens the menu anchored at p, in window pixel coordinates.
func (m *Menu) OpenAt(p f32.Point) {
	m.pos = p
	m.open = true
}

// OpenAtWindow opens the menu at p, in window pixels. Elements given the
// menu with Context open it this way, at the pointer.
func (m *Menu) OpenAtWindow(p image.Point) {
	m.OpenAt(f32.Pt(float32(p.X), float32(p.Y)))
}

// OpenAtClick opens the menu where e happened: at the pointer for a
// right-click or a long-press, below the clicked element for Enter or
// Space on it. A row or a tab opens its context menu with it:
//
//	tab.OnSecondaryClick(menu.OpenAtClick).OnLongPress(menu.OpenAtClick)
func (m *Menu) OpenAtClick(e elements.ClickEvent) {
	m.OpenAt(f32.Pt(float32(e.WindowPosition.X), float32(e.WindowPosition.Y)))
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
