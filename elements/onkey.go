package elements

import (
	"gioui.org/io/key"
	"gioui.org/layout"
)

// keyHandlers are the OnKey handlers registered this frame.
var keyHandlers []func(key.Event) bool

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
func OnKey(fn func(key.Event) bool) {
	keyHandlers = append(keyHandlers, fn)
}

// DispatchKeys delivers the keys left over this frame to the OnKey
// handlers, then clears them. The app (and ui.Frame) calls it once per
// frame, after the portals and DispatchBack; component authors don't
// need to.
func DispatchKeys(gtx layout.Context) {
	handlers := keyHandlers
	keyHandlers = nil
	if len(handlers) == 0 {
		return
	}
	for {
		ev, ok := gtx.Event(key.Filter{Optional: allMods})
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
