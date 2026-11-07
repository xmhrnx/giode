// Package segmented provides a segmented control: a row of mutually
// exclusive options in one rounded track, as iOS and Android use for
// switching views or modes.

package segmented

import (
	"time"

	"gioui.org/layout"
	"gioui.org/widget"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Control is a stateful segmented control. Create it once with New and
// lay it out every frame.
type Control struct {
	// Contextual gives it a context menu (Context).
	elements.Contextual

	options  []string
	icons    []elements.Element
	selected int
	onChange func(i int)
	st       styles.Styles
	clicks   []widget.Clickable
	pass     func(gtx layout.Context) bool

	// pos animates the highlight, in segments.
	pos  float32
	last time.Time
}

// New returns a control with the given options, the first selected.
// The styles argument is optional: Background is the track (default a
// faint tint of the text), BorderColor the selected segment (default
// theme.surface), Color the labels (default theme.text), FontSize, and
// Width: -1 to fill the row with equal segments, or a fixed width they
// share equally.
func New(options ...string) *Control {
	return &Control{options: options, clicks: make([]widget.Clickable, len(options))}
}

// Styles sets the styles. It returns c for chaining.
func (c *Control) Styles(st styles.Styles) *Control {
	c.st = st
	return c
}

// Icons sets an icon per option, shown before its label (or alone for
// an empty label); nil entries have none. It returns c for chaining.
func (c *Control) Icons(icons ...elements.Element) *Control {
	c.icons = icons
	return c
}

// OnChange sets the handler called when the user picks an option. It
// returns c for chaining.
func (c *Control) OnChange(fn func(i int)) *Control {
	c.onChange = fn
	return c
}

// Selected returns the selected option's index.
func (c *Control) Selected() int { return c.selected }

// SetSelected selects option i without invoking OnChange.
func (c *Control) SetSelected(i int) {
	if i >= 0 && i < len(c.options) {
		c.selected = i
	}
}

// Options returns the options.
func (c *Control) Options() []string { return c.options }

// Choose selects option i and invokes OnChange, as a tap would.
func (c *Control) Choose(i int) {
	if i < 0 || i >= len(c.options) || i == c.selected {
		return
	}
	c.selected = i
	if c.onChange != nil {
		c.onChange(i)
	}
}

const animDuration = 160 * time.Millisecond

// trackColor is the default track: a faint tint of the text, mixed
// opaque so it stays faint on dark themes.
var trackColor = properties.Mix("theme.text", "theme.background", 0.08)
