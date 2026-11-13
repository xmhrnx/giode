package base

import (
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
)

// keyHandlers are the OnKey handlers registered this frame; namedKeys
// the keys they asked for by name (see OnKey).
var (
	keyHandlers []func(key.Event) bool
	namedKeys   []NamedKey
	// typing is set while a text field that takes typing has focus.
	typing bool
)

// MarkTyping records, for this frame, that a focused text field takes
// typing: the keys that type text (a character or Space, with no
// modifier but Shift) then don't reach OnKey handlers. Gio sends each
// key both as typed text, to the field, and as a key event, which the
// field doesn't take, so without this "H" or Space typed into a field
// would also run an "H" or "Space" shortcut. A text field calls it
// every frame it has focus; giode's Input does.
func MarkTyping() { typing = true }

// typed reports whether e types text into a field.
func typed(e key.Event) bool {
	if e.Modifiers&^key.ModShift != 0 {
		return false
	}
	if e.Name == key.NameSpace {
		return true
	}
	if gioKeys[e.Name] {
		return false
	}
	r := []rune(string(e.Name))
	return len(r) == 1 && r[0] > ' ' && r[0] != 0x7f
}

// gioKeys are Gio's names for keys that don't type text, which are
// single symbols too.
var gioKeys = map[key.Name]bool{
	key.NameLeftArrow: true, key.NameRightArrow: true, key.NameUpArrow: true, key.NameDownArrow: true,
	key.NameReturn: true, key.NameEnter: true, key.NameEscape: true, key.NameHome: true, key.NameEnd: true,
	key.NameDeleteBackward: true, key.NameDeleteForward: true, key.NamePageUp: true, key.NamePageDown: true,
	key.NameCommand: true,
}

// NamedKey is a key with exactly the modifiers to hold, which an OnKey
// handler asks Gio for by name.
type NamedKey struct {
	Name key.Name
	Mods key.Modifiers
}

// allMods lets an OnKey filter match a key with any modifiers held.
const allMods = key.ModCtrl | key.ModShift | key.ModAlt | key.ModSuper | key.ModCommand

// OnKey registers fn, for this frame, to receive the keys nothing else
// took: app-wide shortcuts. Call it from the view every frame, as with
// HandleBack.
//
// fn sees a key only after everything else in the frame had its
// chance - the focused field, dialogs, menus and other portals, and the
// Back and Escape handlers (HandleBack, HandleDismiss) - so a shortcut
// never steals typing from a text field or Escape from an open dialog,
// whatever order the view lays things out in. While a text field has
// focus, the keys that type into it (a character or Space, alone or
// with Shift) don't reach fn at all (see MarkTyping); shortcuts with
// Ctrl, Cmd or Alt, function keys and Escape do. It gets presses and
// releases (check e.State) and returns whether it used the key. With
// several handlers, the one registered last asks first.
//
// Gio gives some keys - Tab and Shift+Tab, which otherwise move the
// focus - only to a listener that names them. List such keys in named,
// each with exactly the modifiers to hold, for fn to get them; a press
// with other modifiers still moves the focus. Shortcuts does this for
// the keys its shortcuts name.
func OnKey(fn func(key.Event) bool, named ...NamedKey) {
	keyHandlers = append(keyHandlers, fn)
	namedKeys = append(namedKeys, named...)
}

// DispatchKeys delivers the keys left over this frame to the OnKey
// handlers, then clears them. The app (and ui.Frame) calls it once per
// frame, after the portals and DispatchBack; component authors don't
// need to.
func DispatchKeys(gtx layout.Context) {
	handlers, named, typingNow := keyHandlers, namedKeys, typing
	keyHandlers, namedKeys, typing = nil, nil, false
	if len(handlers) == 0 {
		return
	}
	filters := make([]event.Filter, 0, 1+len(named))
	filters = append(filters, key.Filter{Optional: allMods})
	for _, k := range named {
		// Exactly these modifiers: Tab with others held still moves the
		// focus.
		filters = append(filters, key.Filter{Name: k.Name, Required: k.Mods})
	}
	for {
		ev, ok := gtx.Event(filters...)
		if !ok {
			return
		}
		e, ok := ev.(key.Event)
		if !ok || typingNow && typed(e) {
			continue
		}
		for i := len(handlers) - 1; i >= 0; i-- {
			if handlers[i](e) {
				break
			}
		}
	}
}
