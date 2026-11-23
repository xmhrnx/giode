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
	CursorProgress   Cursor = "progress"
	CursorMove       Cursor = "move"
	CursorVertical   Cursor = "vertical-text"
	// Resize cursors, for splitters, panel edges and resize handles:
	// col-resize and row-resize for dividers between columns or rows,
	// the compass ones for edges and corners.
	CursorColResize  Cursor = "col-resize"
	CursorRowResize  Cursor = "row-resize"
	CursorEWResize   Cursor = "ew-resize"
	CursorNSResize   Cursor = "ns-resize"
	CursorNESWResize Cursor = "nesw-resize"
	CursorNWSEResize Cursor = "nwse-resize"
	CursorNResize    Cursor = "n-resize"
	CursorSResize    Cursor = "s-resize"
	CursorEResize    Cursor = "e-resize"
	CursorWResize    Cursor = "w-resize"
	CursorNEResize   Cursor = "ne-resize"
	CursorNWResize   Cursor = "nw-resize"
	CursorSEResize   Cursor = "se-resize"
	CursorSWResize   Cursor = "sw-resize"
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
	case CursorProgress:
		return pointer.CursorProgress
	case CursorMove:
		return pointer.CursorAllScroll
	case CursorVertical:
		return pointer.CursorVerticalText
	case CursorColResize:
		return pointer.CursorColResize
	case CursorRowResize:
		return pointer.CursorRowResize
	case CursorEWResize:
		return pointer.CursorEastWestResize
	case CursorNSResize:
		return pointer.CursorNorthSouthResize
	case CursorNESWResize:
		return pointer.CursorNorthEastSouthWestResize
	case CursorNWSEResize:
		return pointer.CursorNorthWestSouthEastResize
	case CursorNResize:
		return pointer.CursorNorthResize
	case CursorSResize:
		return pointer.CursorSouthResize
	case CursorEResize:
		return pointer.CursorEastResize
	case CursorWResize:
		return pointer.CursorWestResize
	case CursorNEResize:
		return pointer.CursorNorthEastResize
	case CursorNWResize:
		return pointer.CursorNorthWestResize
	case CursorSEResize:
		return pointer.CursorSouthEastResize
	case CursorSWResize:
		return pointer.CursorSouthWestResize
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
