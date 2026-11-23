package terminal

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"gioui.org/io/key"
)

// EncodeKey returns the bytes a terminal sends for a key press that
// doesn't type text, in modes m (the cursor keys' application mode); nil
// for keys it doesn't send. Text arrives separately, as typed.
func EncodeKey(e key.Event, m Modes) []byte {
	if e.State != key.Press {
		return nil
	}
	mods := e.Modifiers
	shift, alt, ctrl := mods.Contain(key.ModShift), mods.Contain(key.ModAlt), mods.Contain(key.ModCtrl)
	// xterm's modifier parameter: 1 + Shift 1, Alt 2, Ctrl 4.
	param := 1
	if shift {
		param += 1
	}
	if alt {
		param += 2
	}
	if ctrl {
		param += 4
	}
	cursor := func(final byte) []byte {
		switch {
		case param > 1:
			return fmt.Appendf(nil, "\x1b[1;%d%c", param, final)
		case m.AppCursor:
			return []byte{0x1b, 'O', final}
		}
		return []byte{0x1b, '[', final}
	}
	tilde := func(n int) []byte {
		if param > 1 {
			return fmt.Appendf(nil, "\x1b[%d;%d~", n, param)
		}
		return fmt.Appendf(nil, "\x1b[%d~", n)
	}
	withAlt := func(b ...byte) []byte {
		if alt {
			return append([]byte{0x1b}, b...)
		}
		return b
	}
	switch e.Name {
	case key.NameUpArrow:
		return cursor('A')
	case key.NameDownArrow:
		return cursor('B')
	case key.NameRightArrow:
		return cursor('C')
	case key.NameLeftArrow:
		return cursor('D')
	case key.NameHome:
		return cursor('H')
	case key.NameEnd:
		return cursor('F')
	case key.NamePageUp:
		return tilde(5)
	case key.NamePageDown:
		return tilde(6)
	case key.NameDeleteForward:
		return tilde(3)
	case key.NameReturn, key.NameEnter:
		return withAlt('\r')
	case key.NameTab:
		if shift {
			return []byte("\x1b[Z")
		}
		return withAlt('\t')
	case key.NameDeleteBackward:
		if ctrl {
			return withAlt(0x08)
		}
		return withAlt(0x7f)
	case key.NameEscape:
		return withAlt(0x1b)
	case key.NameSpace:
		if ctrl {
			return withAlt(0)
		}
		return nil
	case key.NameF1, key.NameF2, key.NameF3, key.NameF4:
		final := "PQRS"[e.Name[1]-'1']
		if param > 1 {
			return fmt.Appendf(nil, "\x1b[1;%d%c", param, final)
		}
		return []byte{0x1b, 'O', final}
	}
	if n, ok := fkeys[e.Name]; ok {
		return tilde(n)
	}
	// Ctrl and Alt with a character key.
	if (ctrl || alt) && !mods.Contain(key.ModCommand) && !mods.Contain(key.ModSuper) {
		r, size := utf8.DecodeRuneInString(string(e.Name))
		if size != len(e.Name) {
			return nil
		}
		if ctrl {
			c, ok := ctrlChar(r)
			if !ok {
				return nil
			}
			return withAlt(c)
		}
		if !shift && r >= 'A' && r <= 'Z' {
			r += 'a' - 'A'
		}
		return withAlt([]byte(string(r))...)
	}
	return nil
}

var fkeys = map[key.Name]int{
	key.NameF5: 15, key.NameF6: 17, key.NameF7: 18, key.NameF8: 19,
	key.NameF9: 20, key.NameF10: 21, key.NameF11: 23, key.NameF12: 24,
}

// ctrlChar is the control character Ctrl with r sends.
func ctrlChar(r rune) (byte, bool) {
	switch {
	case r >= 'A' && r <= 'Z':
		return byte(r - 'A' + 1), true
	case r >= 'a' && r <= 'z':
		return byte(r - 'a' + 1), true
	}
	switch r {
	case '@', '2':
		return 0, true
	case '[', '3':
		return 0x1b, true
	case '\\', '4':
		return 0x1c, true
	case ']', '5':
		return 0x1d, true
	case '^', '6':
		return 0x1e, true
	case '_', '-', '7':
		return 0x1f, true
	case '?', '8':
		return 0x7f, true
	}
	return 0, false
}

// EncodePaste returns pasted text as the terminal sends it: line breaks
// as carriage returns, wrapped in bracketed paste markers when the
// program asked for them (with any markers inside removed, so the text
// can't end the paste early).
func EncodePaste(text string, m Modes) []byte {
	text = strings.ReplaceAll(text, "\r\n", "\r")
	text = strings.ReplaceAll(text, "\n", "\r")
	if !m.BracketedPaste {
		return []byte(text)
	}
	text = strings.ReplaceAll(text, "\x1b[201~", "")
	return []byte("\x1b[200~" + text + "\x1b[201~")
}

// MouseButton is a mouse button for EncodeMouse.
type MouseButton uint8

const (
	MouseLeft MouseButton = iota
	MouseMiddle
	MouseRight
	MouseNone // a move with no button held
	MouseWheelUp
	MouseWheelDown
)

// MouseAction is what a mouse event did.
type MouseAction uint8

const (
	MousePress MouseAction = iota
	MouseRelease
	MouseMove
)

// EncodeMouse returns the bytes reporting a mouse event at cell (x, y),
// counting from 0, to a program that asked for mouse events (Modes.Mouse);
// nil when it didn't ask for this kind.
func EncodeMouse(m Modes, b MouseButton, a MouseAction, x, y int, mods key.Modifiers) []byte {
	switch m.Mouse {
	case MouseOff:
		return nil
	case MouseClicks:
		if a == MouseMove {
			return nil
		}
	case MouseDrags:
		if a == MouseMove && b == MouseNone {
			return nil
		}
	}
	code := 0
	switch b {
	case MouseLeft, MouseMiddle, MouseRight:
		code = int(b)
	case MouseNone:
		code = 3
	case MouseWheelUp:
		code = 64
	case MouseWheelDown:
		code = 65
	}
	if a == MouseMove {
		code += 32
	}
	if mods.Contain(key.ModShift) {
		code += 4
	}
	if mods.Contain(key.ModAlt) {
		code += 8
	}
	if mods.Contain(key.ModCtrl) {
		code += 16
	}
	if m.MouseSGR {
		final := 'M'
		if a == MouseRelease {
			final = 'm'
		}
		return fmt.Appendf(nil, "\x1b[<%d;%d;%d%c", code, x+1, y+1, final)
	}
	if a == MouseRelease {
		code = code&^3 | 3
	}
	if x > 222 || y > 222 {
		// The classic encoding can't say it.
		return nil
	}
	return []byte{0x1b, '[', 'M', byte(32 + code), byte(33 + x), byte(33 + y)}
}

// EncodeFocus returns the report of the terminal gaining or losing focus,
// when the program asked for them.
func EncodeFocus(m Modes, focused bool) []byte {
	if !m.FocusEvents {
		return nil
	}
	if focused {
		return []byte("\x1b[I")
	}
	return []byte("\x1b[O")
}
