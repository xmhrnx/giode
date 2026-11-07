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
)

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
// whatever order the view lays things out in. It gets presses and
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
	handlers, named := keyHandlers, namedKeys
	keyHandlers, namedKeys = nil, nil
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
		if !ok {
			continue
		}
		for i := len(handlers) - 1; i >= 0; i-- {
			if handlers[i](e) {
				break
			}
		}
	}
}
