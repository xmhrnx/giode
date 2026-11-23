package base

import (
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
)

// keyHandlers are the OnKey handlers registered this frame; namedKeys
// the keys they asked for by name (see OnKey).
var (
	keyHandlers []func(key.Event) bool
	namedKeys   []NamedKey
	// typing is set while a text field that takes typing has focus.
	typing bool
	// keyWatchers are the WatchKeys handlers registered this frame;
	// seenKeys the keys components took this frame, for them.
	keyWatchers []func(key.Event)
	seenKeys    []key.Event
	// forwarded are keys a component took and passed on to the OnKey
	// handlers (ForwardKey).
	forwarded []key.Event
)

// WatchKeys registers fn, for this frame, to see every key press and
// release the app gets, before anything acts on it: the keys nothing
// took, which then go on to the OnKey handlers, and the ones a focused
// component took (a KeyboardInput's, a button's Enter and Space,
// an Input's OnKeys). Call it from the view every frame, as with OnKey.
// fn only watches: it can't stop a key from going where it goes. It
// sees the modifier keys themselves (key.NameShift, NameCtrl, NameAlt,
// NameCommand, NameSuper) pressed and released, so it can spot a
// double Shift. The keys a focused text field (Input) edits with -
// arrows, Backspace, Delete, Enter, Home, End and its Ctrl or Cmd
// shortcuts - stay inside the field and aren't seen; its typed text
// isn't a key press either. The keys of a frame reach fn at the end of
// it.
func WatchKeys(fn func(key.Event)) {
	keyWatchers = append(keyWatchers, fn)
}

// SeenKey reports a key a component took, for the WatchKeys handlers.
// Components that take keys from Gio themselves call it; app code
// doesn't need to.
func SeenKey(e key.Event) {
	seenKeys = append(seenKeys, e)
}

// ForwardKey passes a key a component took but didn't use on to the
// OnKey handlers (app shortcuts) at the end of the frame, as if nothing
// had taken it; the component reports it with SeenKey as well.
func ForwardKey(e key.Event) {
	forwarded = append(forwarded, e)
}

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
	watchers, seen, fwd := keyWatchers, seenKeys, forwarded
	keyHandlers, namedKeys, typing = nil, nil, false
	keyWatchers, seenKeys, forwarded = nil, nil, nil
	watch := func(e key.Event) {
		for _, w := range watchers {
			w(e)
		}
	}
	for _, e := range seen {
		watch(e)
	}
	dispatch := func(e key.Event) {
		for i := len(handlers) - 1; i >= 0; i-- {
			if handlers[i](e) {
				// The frame is laid out already: what the handler
				// changed shows in the next one, which nothing else
				// would ask for until the next input.
				gtx.Execute(op.InvalidateCmd{})
				break
			}
		}
	}
	for _, e := range fwd {
		dispatch(e)
	}
	if len(handlers) == 0 && len(watchers) == 0 {
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
		watch(e)
		if typingNow && typed(e) {
			continue
		}
		dispatch(e)
	}
}
