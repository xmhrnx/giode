// Package tooltip provides a stateful tooltip, shown on hover or
// long-press, that wraps any element as its trigger.

package tooltip

import (
	"time"

	"gioui.org/layout"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
)

// Tooltip is a stateful tooltip wrapping trigger: hovering trigger
// with a mouse, or long-pressing it on a touch screen, shows content
// in a small panel anchored just below it, rendered above everything
// else via elements.RegisterPortal. After a long-press the tooltip
// stays for a moment once the finger lifts. Create it once with New
// and lay it out every frame in trigger's place.
type Tooltip struct {
	pass    func(gtx layout.Context) bool
	trigger elements.Element
	content string
	st      styles.Styles
	// placement is the side of the trigger the panel goes.
	placement elements.Placement

	// hovered is set while a mouse is over the trigger.
	hovered bool
	// touchAt is when a finger went down on the trigger; zero when
	// none is down.
	touchAt time.Time
	// longPressed is set once that finger rested for longPress.
	longPressed bool
	// shownUntil keeps the tooltip visible after a long-press ends.
	shownUntil time.Time
}

var (
	defaultBackground = "#0f172a"
	defaultText       = "#f8fafc"
)

// Touch timings: how long a finger must rest on the trigger to show
// the tooltip, and how long it stays after the finger lifts.
const (
	longPress = 500 * time.Millisecond
	linger    = 1500 * time.Millisecond
)

// New wraps trigger with a tooltip showing content on hover or
// long-press. The styles argument is optional: Background and Color
// style the panel, FontSize its text.
func New(trigger elements.Element, content string, st ...styles.Styles) *Tooltip {
	t := &Tooltip{trigger: trigger, content: content}
	if len(st) > 0 {
		t.st = st[0]
	}
	return t
}

// Placement chooses the side of the trigger the tooltip shows on:
// elements.PlaceAuto (the default) shows it below, or above when it
// wouldn't fit below; PlaceBelow and PlaceAbove keep it on one side.
func (t *Tooltip) Placement(p elements.Placement) *Tooltip {
	t.placement = p
	return t
}

// SetContent replaces the tooltip text.
func (t *Tooltip) SetContent(content string) {
	t.content = content
}

// Hovered reports whether a mouse was over the trigger on the last
// frame.
func (t *Tooltip) Hovered() bool {
	return t.hovered
}

// Visible reports whether the tooltip was showing on the last frame,
// from hover or a long-press.
func (t *Tooltip) Visible() bool {
	return t.hovered || t.longPressed || !t.shownUntil.IsZero()
}
