package elements

import "image"

// KeyboardOwner is the focused text field, for dismissing the software
// keyboard with a tap outside it: the window asks it whether a tap
// belongs to it.
type KeyboardOwner interface {
	// KeepsKeyboard reports whether a tap at p, in window coordinates,
	// lands on the field or its own overlays (its edit menu and
	// selection handles).
	KeepsKeyboard(p image.Point) bool
}

var keyboardOwner KeyboardOwner

// SetKeyboardOwner records the focused text field for this frame. A
// focused field calls it every frame it's laid out.
func SetKeyboardOwner(o KeyboardOwner) { keyboardOwner = o }

// TakeKeyboardOwner returns the field recorded this frame, if any, and
// clears it for the next.
func TakeKeyboardOwner() KeyboardOwner {
	o := keyboardOwner
	keyboardOwner = nil
	return o
}
