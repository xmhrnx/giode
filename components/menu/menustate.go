// Package menu provides a right-click context menu that opens at the
// cursor and keeps itself within the bounds of the window, flipping
// above the cursor when it would overflow the bottom edge.

package menu

import (
	"gioui.org/f32"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Menu is a stateful context menu. Register it with App.SetContextMenu
// to open it with a right-click anywhere in the window, call
// OpenAt/Toggle directly, or wrap a trigger element with Trigger for a
// click-to-open dropdown menu.
//
// When SetContextMenu is used the app registers the right-click area
// and renders the menu on top automatically; otherwise lay Menu out
// anywhere in the view (its panel renders via elements.RegisterPortal,
// so it paints above everything else regardless of tree position).
type Menu struct {
	items  []Item
	open   bool
	pos    f32.Point
	styles styles.Styles
	// itemPadding overrides the rows' padding when itemPaddingSet.
	itemPadding    properties.Inset
	itemPaddingSet bool
	// hoverColor and separatorColor override the defaults when set.
	hoverColor, separatorColor, disabledColor properties.Color

	// openSubmenu is the index of the item whose submenu is showing,
	// or -1. It persists once hover opens a submenu - set again every
	// frame while that item stays the active one - rather than closing
	// the moment the pointer leaves the row, so moving into the
	// submenu's own (separately positioned) panel doesn't flicker it
	// shut; it only changes when a sibling submenu item is hovered, or
	// this menu closes.
	openSubmenu int
	// parent is the menu a submenu opens from (nil for a root menu):
	// choosing an item in a submenu closes the whole menu.
	parent *Menu
	// panelLeft is the left edge of the panel in the frame being laid
	// out, in window pixels; opensLeft records that a submenu opened to
	// the left of its parent, which its own submenus then prefer too.
	panelLeft int
	opensLeft bool
	// isSubmenu marks a Menu opened via SubMenu rather than
	// SetContextMenu/OpenAt/Trigger: its visibility is driven entirely
	// by the parent row's hover (openSubmenu above), so it skips its
	// own dismiss-on-outside-click scrim and Back/Escape handling, which
	// would otherwise need to cover the whole window from deep inside
	// the parent panel's own (much smaller) layout constraints.
	isSubmenu bool

	// scrim is the tag of the window-wide area that closes the menu on
	// a press of any button outside it.
	scrim bool

	// scroll scrolls rows that don't fit in the window (scroll.go).
	scroll scrollState
	// active is the item the keyboard highlights, or -1; keyNav (on
	// the root menu) records that the keyboard moved last, so the
	// highlight shows; keyboardIn marks a submenu the keyboard entered.
	active     int
	keyNav     bool
	keyboardIn bool
	// keyTag is a root menu's keyboard focus target; hasFocus records
	// that it asked for focus since opening (keys.go).
	keyTag   bool
	hasFocus bool
}

// root returns the menu at the top of a chain of submenus.
func (m *Menu) root() *Menu {
	for m.parent != nil {
		m = m.parent
	}
	return m
}

// New returns an empty menu. The styles argument is optional:
// Background, BorderRadius, Padding and Color (item text) are
// honored; the rest fall back to defaults.
func New(st ...styles.Styles) *Menu {
	m := &Menu{openSubmenu: -1, active: -1}
	if len(st) > 0 {
		m.styles = st[0]
	}
	return m
}

// hasLeading reports whether any item has an icon or a check, so every
// row reserves the leading column and the labels line up.
func (m *Menu) hasLeading() bool {
	for i := range m.items {
		if m.items[i].icon != nil || m.items[i].check != checkNone {
			return true
		}
	}
	return false
}

// ItemPadding sets the padding of every row, in dp, replacing the
// default of 28 horizontal and 8 vertical. A submenu has its own
// setting. It returns m for chaining.
func (m *Menu) ItemPadding(p properties.Inset) *Menu {
	m.itemPadding = p
	m.itemPaddingSet = true
	return m
}

// HoverColor sets the background of the row under the pointer. It
// returns m for chaining.
func (m *Menu) HoverColor(c properties.Color) *Menu {
	m.hoverColor = c
	return m
}

// SeparatorColor sets the color of Separator lines. It returns m for
// chaining.
func (m *Menu) SeparatorColor(c properties.Color) *Menu {
	m.separatorColor = c
	return m
}

// DisabledColor sets the text color of DisabledItem rows. It returns m
// for chaining.
func (m *Menu) DisabledColor(c properties.Color) *Menu {
	m.disabledColor = c
	return m
}

// rowPadding is the padding of each row.
func (m *Menu) rowPadding() properties.Inset {
	if m.itemPaddingSet {
		return m.itemPadding
	}
	return properties.SymmetricInset(28, 8)
}
