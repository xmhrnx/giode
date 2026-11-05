// Package sheet provides a modal sheet: a panel that slides up from the
// bottom of the window on phones and, with Adaptive, shows as a
// centered dialog on wide screens.

package sheet

import (
	"time"
)

// Detent is a height a bottom sheet can rest at.
type Detent int

const (
	// DetentFit sizes the sheet to its content, up to nearly the full
	// window (the default).
	DetentFit Detent = iota
	// DetentMedium is half the window.
	DetentMedium
	// DetentLarge is nearly the full window.
	DetentLarge
)

const animDuration = 220 * time.Millisecond

// adaptiveWidth is the window width, in dp, from which an Adaptive
// sheet shows as a centered dialog.
const adaptiveWidth = 600
