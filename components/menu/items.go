// Package menu provides a right-click context menu that opens at the
// cursor and keeps itself within the bounds of the window, flipping
// above the cursor when it would overflow the bottom edge.

package menu

import (
	"gioui.org/widget"
	"github.com/xmhrnx/giode/components/icon"
)

// Item is a single row of a menu.
type Item struct {
	label     string
	icon      icon.Source
	disabled  bool
	separator bool
	click     *widget.Clickable
	action    func()
	submenu   *Menu
	// shortcut is shown right-aligned ("Ctrl+O").
	shortcut string
	// check is checkNone, checkBox or checkRadio; checked reports the
	// item's state each frame.
	check   int
	checked func() bool
}

const (
	checkNone = iota
	checkBox
	checkRadio
)

// Item appends a clickable item. It returns m for chaining.
func (m *Menu) Item(label string, action func()) *Menu {
	m.items = append(m.items, Item{
		label:  label,
		click:  new(widget.Clickable),
		action: action,
	})
	return m
}

// IconItem appends an item with a leading icon - any icon.Source: a
// loaded SVG document, a registered icon (icon.Lookup) or path data
// (icon.Path). It returns m for chaining.
func (m *Menu) IconItem(ic icon.Source, label string, action func()) *Menu {
	m.items = append(m.items, Item{
		label:  label,
		icon:   ic,
		click:  new(widget.Clickable),
		action: action,
	})
	return m
}

// CheckItem appends an item showing a check mark while checked reports
// true - the state lives in the app and is read every frame. action
// runs on click; it's up to it to flip the state. It returns m for
// chaining.
func (m *Menu) CheckItem(label string, checked func() bool, action func()) *Menu {
	m.Item(label, action)
	m.items[len(m.items)-1].check, m.items[len(m.items)-1].checked = checkBox, checked
	return m
}

// RadioItem appends an item showing a dot while selected reports true:
// one of a group of choices (a playback speed, an audio track), where
// each item's action selects it. It returns m for chaining.
func (m *Menu) RadioItem(label string, selected func() bool, action func()) *Menu {
	m.Item(label, action)
	m.items[len(m.items)-1].check, m.items[len(m.items)-1].checked = checkRadio, selected
	return m
}

// Shortcut sets the keyboard shortcut shown right-aligned on the last
// item added: m.Item("Open File…", open).Shortcut("Mod+O"). A spec the
// shortcut package parses shows the way the platform writes it ("⌘O"
// on macOS, "Ctrl+O" elsewhere); other text shows as it is. It only
// shows the shortcut: register the key with giode.Shortcuts. It
// returns m for chaining.
func (m *Menu) Shortcut(text string) *Menu {
	if n := len(m.items); n > 0 {
		m.items[n-1].shortcut = text
	}
	return m
}

// DisabledItem appends a grayed-out, non-clickable item. It returns m
// for chaining.
func (m *Menu) DisabledItem(label string) *Menu {
	m.items = append(m.items, Item{label: label, disabled: true})
	return m
}

// Separator appends a horizontal line. It returns m for chaining.
func (m *Menu) Separator() *Menu {
	m.items = append(m.items, Item{separator: true})
	return m
}

// SubMenu appends an item that reveals submenu as a nested panel,
// cascading to the side, when hovered. submenu is a *Menu of its own
// - built the same way as any other (Item, IconItem, Separator,
// SubMenu for further nesting) - but must not be registered with
// SetContextMenu, OpenAt, Toggle or Trigger itself: this item's hover
// state is its only way of opening. It returns m for chaining.
func (m *Menu) SubMenu(label string, submenu *Menu) *Menu {
	submenu.isSubmenu = true
	submenu.parent = m
	m.items = append(m.items, Item{
		label:   label,
		click:   new(widget.Clickable),
		submenu: submenu,
	})
	return m
}

// IconSubMenu is SubMenu with a leading icon, so it lines up with
// IconItem rows. It returns m for chaining.
func (m *Menu) IconSubMenu(ic icon.Source, label string, submenu *Menu) *Menu {
	m.SubMenu(label, submenu)
	m.items[len(m.items)-1].icon = ic
	return m
}
