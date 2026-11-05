package elements

import (
	"github.com/xmhrnx/giode/elements/box"
)

// Box and Pressable, re-exported from elements/box.
type (
	// BoxElement is a container element: a flex container and a
	// containing block for absolutely positioned children.
	BoxElement = box.BoxElement
	// Pressable makes any content clickable.
	Pressable = box.Pressable
	// PressState is the interaction state a Pressable's content is
	// built with.
	PressState = box.PressState
	// MouseButton identifies which button produced a click.
	MouseButton = box.MouseButton
	// ClickSource says how a click was produced.
	ClickSource = box.ClickSource
	// ClickEvent describes a click.
	ClickEvent = box.ClickEvent
)

const (
	// LongPressDuration is how long a touch must be held still for
	// OnLongPress.
	LongPressDuration = box.LongPressDuration

	MousePrimary   = box.MousePrimary
	MouseSecondary = box.MouseSecondary
	MouseMiddle    = box.MouseMiddle

	// ClickPointer is a mouse, pen or touch click.
	ClickPointer = box.ClickPointer
	// ClickKeyboard is Enter or Space on a focused Pressable.
	ClickKeyboard = box.ClickKeyboard
	// ClickProgrammatic is a call to Pressable.Click (or automation).
	ClickProgrammatic = box.ClickProgrammatic
)

var (
	Box                  = box.Box
	Row                  = box.Row
	Stack                = box.Stack
	NewPressable         = box.NewPressable
	SetLongPressFeedback = box.SetLongPressFeedback
)
