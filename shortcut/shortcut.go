// Package shortcut parses keyboard shortcuts written as text ("Mod+T",
// "Ctrl+Shift+P", "F5", "Mod+,"), matches them against key events and
// formats them for display the way each platform writes them.
//
// "Mod" is the platform's shortcut modifier: Command on macOS and iOS,
// Ctrl elsewhere. The other modifiers are Ctrl, Shift, Alt (or Option)
// and Super (or Cmd, Win, Meta). The key is a letter or digit, a
// punctuation character, F1-F12, or one of Enter, Esc, Tab, Space,
// Backspace, Delete, Home, End, PageUp, PageDown, Up, Down, Left and
// Right, in any case.
package shortcut

import (
	"fmt"
	"runtime"
	"strings"

	"gioui.org/io/key"
)

// Shortcut is a key with its modifiers and the action it runs.
type Shortcut struct {
	// Spec is the text the shortcut was parsed from ("Mod+T").
	Spec   string
	Name   key.Name
	Mods   key.Modifiers
	Action func()
}

// New parses spec and returns the shortcut running action. A spec that
// doesn't parse panics, since it's a mistake in the program; use Parse
// for user-entered shortcuts.
func New(spec string, action func()) Shortcut {
	name, mods, err := Parse(spec)
	if err != nil {
		panic(err)
	}
	return Shortcut{Spec: spec, Name: name, Mods: mods, Action: action}
}

// Matches reports whether e is a press of the shortcut's key with
// exactly its modifiers.
func (s Shortcut) Matches(e key.Event) bool {
	return e.State == key.Press && e.Name == s.Name && e.Modifiers == s.Mods
}

// Handle runs the first shortcut matching e and reports whether one
// did. It's a giode.OnKey handler for a list of shortcuts.
func Handle(e key.Event, list []Shortcut) bool {
	for _, s := range list {
		if s.Matches(e) {
			if s.Action != nil {
				s.Action()
			}
			return true
		}
	}
	return false
}

var modNames = map[string]key.Modifiers{
	"mod": key.ModShortcut, "ctrl": key.ModCtrl, "control": key.ModCtrl,
	"shift": key.ModShift, "alt": key.ModAlt, "option": key.ModAlt, "opt": key.ModAlt,
	"super": key.ModSuper, "win": key.ModSuper, "meta": key.ModSuper,
	"cmd": key.ModCommand, "command": key.ModCommand,
}

var keyNames = map[string]key.Name{
	"enter": key.NameReturn, "return": key.NameReturn, "esc": key.NameEscape, "escape": key.NameEscape,
	"tab": key.NameTab, "space": key.NameSpace, "backspace": key.NameDeleteBackward,
	"delete": key.NameDeleteForward, "del": key.NameDeleteForward, "home": key.NameHome, "end": key.NameEnd,
	"pageup": key.NamePageUp, "pagedown": key.NamePageDown, "pgup": key.NamePageUp, "pgdn": key.NamePageDown,
	"up": key.NameUpArrow, "down": key.NameDownArrow, "left": key.NameLeftArrow, "right": key.NameRightArrow,
}

// Parse reads a shortcut spec into Gio's key name and modifiers.
func Parse(spec string) (key.Name, key.Modifiers, error) {
	s := strings.TrimSpace(spec)
	if s == "" {
		return "", 0, fmt.Errorf("shortcut: empty")
	}
	// The key may itself be "+": "Mod++" or a lone "+".
	var parts []string
	if strings.HasSuffix(s, "++") || s == "+" {
		parts = append(strings.Split(strings.TrimSuffix(strings.TrimSuffix(s, "+"), "+"), "+"), "+")
		if s == "+" {
			parts = []string{"+"}
		}
	} else {
		parts = strings.Split(s, "+")
	}
	var mods key.Modifiers
	for _, p := range parts[:len(parts)-1] {
		m, ok := modNames[strings.ToLower(strings.TrimSpace(p))]
		if !ok {
			return "", 0, fmt.Errorf("shortcut %q: unknown modifier %q", spec, p)
		}
		mods |= m
	}
	k := strings.TrimSpace(parts[len(parts)-1])
	switch lk := strings.ToLower(k); {
	case k == "":
		return "", 0, fmt.Errorf("shortcut %q: no key", spec)
	case keyNames[lk] != "":
		return keyNames[lk], mods, nil
	case len(lk) >= 2 && lk[0] == 'f' && isNumber(lk[1:]):
		return key.Name(strings.ToUpper(k)), mods, nil
	case len([]rune(k)) == 1:
		return key.Name(strings.ToUpper(k)), mods, nil
	}
	return "", 0, fmt.Errorf("shortcut %q: unknown key %q", spec, k)
}

func isNumber(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}

// Format writes spec for display on this platform: "⇧⌘T" on macOS and
// iOS, "Ctrl+Shift+T" elsewhere. A spec that doesn't parse is returned
// as it is.
func Format(spec string) string {
	return format(spec, runtime.GOOS == "darwin" || runtime.GOOS == "ios")
}

var display = map[key.Name]string{
	key.NameReturn: "Enter", key.NameEscape: "Esc", key.NameDeleteBackward: "Backspace",
	key.NameDeleteForward: "Delete", key.NameHome: "Home", key.NameEnd: "End",
	key.NamePageUp: "PageUp", key.NamePageDown: "PageDown", key.NameUpArrow: "↑",
	key.NameDownArrow: "↓", key.NameLeftArrow: "←", key.NameRightArrow: "→",
}

var macDisplay = map[key.Name]string{
	key.NameReturn: "↩", key.NameEscape: "⎋", key.NameDeleteBackward: "⌫", key.NameDeleteForward: "⌦",
	key.NameTab: "⇥", key.NameSpace: "Space", key.NamePageUp: "⇞", key.NamePageDown: "⇟",
	key.NameHome: "↖", key.NameEnd: "↘",
}

func format(spec string, mac bool) string {
	name, mods, err := Parse(spec)
	if err != nil {
		return spec
	}
	k := string(name)
	if d, ok := display[name]; ok {
		k = d
	}
	if mac {
		if d, ok := macDisplay[name]; ok {
			k = d
		}
		var b strings.Builder
		for _, m := range []struct {
			mod key.Modifiers
			sym string
		}{{key.ModCtrl, "⌃"}, {key.ModAlt, "⌥"}, {key.ModShift, "⇧"}, {key.ModCommand, "⌘"}, {key.ModSuper, "⌘"}} {
			if mods.Contain(m.mod) {
				b.WriteString(m.sym)
			}
		}
		return b.String() + k
	}
	var parts []string
	for _, m := range []struct {
		mod  key.Modifiers
		name string
	}{{key.ModCtrl, "Ctrl"}, {key.ModCommand, "Cmd"}, {key.ModSuper, "Win"}, {key.ModAlt, "Alt"}, {key.ModShift, "Shift"}} {
		if mods.Contain(m.mod) {
			parts = append(parts, m.name)
		}
	}
	return strings.Join(append(parts, k), "+")
}
