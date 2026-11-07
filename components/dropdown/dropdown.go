// Package dropdown provides a stateful dropdown.
package dropdown

import (
	"image"
	"image/color"

	"gioui.org/widget"

	"github.com/xmhrnx/giode/components/scroll"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

var (
	defaultBackground = color.NRGBA{R: 0x1e, G: 0x29, B: 0x3b, A: 0xff}
	defaultText       = color.NRGBA{R: 0xe2, G: 0xe8, B: 0xf0, A: 0xff}
	defaultBorder     = color.NRGBA{R: 0x47, G: 0x55, B: 0x69, A: 0xff}
	defaultHover      = color.NRGBA{R: 0x33, G: 0x41, B: 0x55, A: 0xff}
)

// Select is a stateful dropdown. Create it once with New and lay it
// out every frame.
type Select struct {
	// Contextual gives it a context menu (Context).
	elements.Contextual

	options   []string
	selected  int
	open      bool
	field     widget.Clickable
	itemClick []*widget.Clickable
	onChange  func(index int)
	st        styles.Styles
	// hover overrides the option hover background when set.
	hover properties.Color

	sheetMode SheetMode
	sheet     sheetState
	// outside is the target of presses outside the open list.
	outside struct{}
	// list scrolls the options when they don't fit beside the field;
	// reveal scrolls the selected one into view once the list opens.
	list *scroll.Scroll
	// at is the field's window position, tracked as containers place it.
	at     image.Point
	reveal bool
}

// New returns a Select with the given styles and options: Background
// is the dropdown background, Color the text, BorderColor the field
// border, BorderRadius the corners.
func New(st styles.Styles, options ...string) *Select {
	s := &Select{options: options, st: st}
	for range options {
		s.itemClick = append(s.itemClick, new(widget.Clickable))
	}
	return s
}

// HoverColor sets the background of the option under the pointer. It
// returns s for chaining.
func (s *Select) HoverColor(c properties.Color) *Select {
	s.hover = c
	return s
}

// Selected returns the index of the selected option.
func (s *Select) Selected() int {
	return s.selected
}

// Value returns the selected option's text.
func (s *Select) Value() string {
	if s.selected >= len(s.options) {
		return ""
	}
	return s.options[s.selected]
}

// SetSelected changes the selection without invoking OnChange.
func (s *Select) SetSelected(i int) {
	if i >= 0 && i < len(s.options) {
		s.selected = i
	}
}

// Options returns the selectable options.
func (s *Select) Options() []string {
	return s.options
}

// SetOptions replaces the options. The selection stays on the same text
// when it's still among them, and otherwise moves to the first option;
// OnChange isn't called. An open list shows the new options.
func (s *Select) SetOptions(options ...string) {
	cur := s.Value()
	s.options = options
	for len(s.itemClick) < len(options) {
		s.itemClick = append(s.itemClick, new(widget.Clickable))
	}
	s.selected = 0
	for i, o := range options {
		if o == cur {
			s.selected = i
			break
		}
	}
}

// Choose selects option i and invokes OnChange, as picking it from the
// list would. Out-of-range indexes are ignored.
func (s *Select) Choose(i int) {
	if i < 0 || i >= len(s.options) {
		return
	}
	s.selected = i
	s.open = false
	if s.onChange != nil {
		s.onChange(i)
	}
}

// OnChange sets the handler invoked when the selection changes. It
// returns s for chaining.
func (s *Select) OnChange(fn func(index int)) *Select {
	s.onChange = fn
	return s
}

// Opened reports whether the dropdown is open.
func (s *Select) Opened() bool {
	return s.open
}
