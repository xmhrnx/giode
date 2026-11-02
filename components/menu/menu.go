// Package menu provides a right-click context menu that opens at the
// cursor and keeps itself within the bounds of the window, flipping
// above the cursor when it would overflow the bottom edge.
package menu

import (
	"gioui.org/f32"
	"gioui.org/widget"

	"github.com/xmhrnx/giode/components/icon"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
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
