package input

import (
	"gioui.org/io/key"
)

// Keyboard selects the on-screen keyboard a field asks for on touch
// devices. Desktop platforms ignore it.
type Keyboard uint8

const (
	// KeyboardText is the plain text keyboard, the default.
	KeyboardText Keyboard = iota
	// KeyboardNumeric shows digits, "." and ",".
	KeyboardNumeric
	// KeyboardEmail adds shortcuts such as "@" and ".com".
	KeyboardEmail
	// KeyboardURL adds shortcuts such as "/" and ".com".
	KeyboardURL
	// KeyboardPhone shows digits, "#" and "*".
	KeyboardPhone
)

// hint maps k to Gio's keyboard hint.
func (k Keyboard) hint() key.InputHint {
	switch k {
	case KeyboardNumeric:
		return key.HintNumeric
	case KeyboardEmail:
		return key.HintEmail
	case KeyboardURL:
		return key.HintURL
	case KeyboardPhone:
		return key.HintTelephone
	}
	return key.HintAny
}

// SubmitMode selects which key presses submit a field.
type SubmitMode int

const (
	// SubmitDefault submits on Enter in a single-line field; a
	// multiline field inserts a newline and never submits.
	SubmitDefault SubmitMode = iota
	// SubmitEnter submits on Enter, also in a multiline field.
	SubmitEnter
	// SubmitEnterUnlessShift submits on Enter and inserts a newline on
	// Shift+Enter, as chat composers do. On a touch screen the
	// on-screen keyboard's Enter key inserts a newline instead, since
	// phone apps send with a button.
	SubmitEnterUnlessShift
	// SubmitNever never submits; Enter inserts a newline in a
	// multiline field.
	SubmitNever
)
