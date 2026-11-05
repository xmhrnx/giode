package box

import (
	"image"
	"time"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/layout"
)

// PressState is the interaction state a Pressable's content is built
// with, so it can restyle itself.
type PressState struct {
	Hovered  bool
	Pressed  bool
	Focused  bool
	Disabled bool
}

// ClickEvent describes a click.
type ClickEvent struct {
	// Clicks is 1 for a single click, 2 for a double click, and so on.
	Clicks int
	// Button is the mouse button; MousePrimary for keyboard and
	// programmatic clicks.
	Button MouseButton
	// Modifiers holds the keys held down (key.ModCtrl, key.ModShift,
	// ...); see Ctrl and Shift.
	Modifiers key.Modifiers
	// Source says how the click was produced.
	Source ClickSource
	// Position is where the pointer was, relative to the Pressable,
	// in pixels; zero for keyboard and programmatic clicks.
	Position image.Point
	// PositionDp is Position in dp, the unit of Styles.
	PositionDp f32.Point
	// WindowPosition is where the pointer was in the window, in pixels:
	// what Menu.OpenAt takes, so a row can open a menu where it was
	// right-clicked.
	WindowPosition image.Point
}

// MouseButton identifies which button produced a click.
type MouseButton int

const (
	MousePrimary MouseButton = iota
	MouseSecondary
	MouseMiddle
)

// ClickSource says how a click was produced.
type ClickSource int

const (
	// ClickPointer is a mouse, pen or touch click.
	ClickPointer ClickSource = iota
	// ClickKeyboard is Enter or Space on a focused Pressable.
	ClickKeyboard
	// ClickProgrammatic is a call to Pressable.Click (or automation).
	ClickProgrammatic
)

// toDp converts a pixel position to dp.
func toDp(pt f32.Point, gtx layout.Context) f32.Point {
	if s := gtx.Metric.PxPerDp; s > 0 {
		return pt.Div(s)
	}
	return pt
}

// Ctrl reports whether Ctrl (or Command on macOS) was held.
func (e ClickEvent) Ctrl() bool {
	return e.Modifiers.Contain(key.ModShortcut)
}

// Shift reports whether Shift was held.
func (e ClickEvent) Shift() bool {
	return e.Modifiers.Contain(key.ModShift)
}

// multiClickWindow is the longest gap between clicks that still counts
// them as a double (or triple) click, and multiClickSlop how far apart
// they may be, in dp.
const (
	multiClickWindow = 400 * time.Millisecond
	multiClickSlop   = 6
)
