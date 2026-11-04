package properties

import (
	"runtime"

	"gioui.org/io/pointer"
)

// Cursor is the cursor shown while hovering an interactive component
// (CSS cursor).
type Cursor string

const (
	CursorDefault    Cursor = "default"
	CursorNone       Cursor = "none"
	CursorText       Cursor = "text"
	CursorPointer    Cursor = "pointer"
	CursorCrosshair  Cursor = "crosshair"
	CursorGrab       Cursor = "grab"
	CursorGrabbing   Cursor = "grabbing"
	CursorNotAllowed Cursor = "not-allowed"
	CursorWait       Cursor = "wait"
)

// CursorFor maps the value to a pointer cursor, defaulting to the
// system default.
func (c Cursor) CursorFor() pointer.Cursor {
	switch c {
	case CursorNone:
		return pointer.CursorNone
	case CursorText:
		return pointer.CursorText
	case CursorPointer:
		return pointer.CursorPointer
	case CursorCrosshair:
		return pointer.CursorCrosshair
	case CursorGrab:
		return grabCursor(pointer.CursorGrab)
	case CursorGrabbing:
		return grabCursor(pointer.CursorGrabbing)
	case CursorNotAllowed:
		return pointer.CursorNotAllowed
	case CursorWait:
		return pointer.CursorWait
	default:
		return pointer.CursorDefault
	}
}

// grabCursor is c, the open or closed hand, where the system has one.
// Windows has none, and Gio shows its four-way move arrow instead,
// which reads as "move the window": the pointing hand is used there.
func grabCursor(c pointer.Cursor) pointer.Cursor {
	if runtime.GOOS == "windows" {
		return pointer.CursorPointer
	}
	return c
}
