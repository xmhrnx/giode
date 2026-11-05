// Package sheet provides a modal sheet: a panel that slides up from the
// bottom of the window on phones and, with Adaptive, shows as a
// centered dialog on wide screens.

package sheet

import (
	"time"

	"gioui.org/gesture"
	"gioui.org/widget"
	"github.com/xmhrnx/giode/components/scroll"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
)

// Sheet is a modal panel over a dimmed scrim: a bottom sheet that
// slides up from the bottom edge, or with Adaptive a centered dialog on
// wide windows. It has an optional title, a scrolling body and a
// footer pinned below it. The scrim, Back and Escape close it, as does
// dragging its handle down; with several Detents, dragging moves it
// between them.
//
// Like Drawer it takes no space in the layout and paints over the
// window while open, so it can sit anywhere in the tree. Create it
// once with New, outside the view function.
type Sheet struct {
	open     *bool
	content  func() elements.Element
	footer   func() elements.Element
	title    string
	st       styles.Styles
	detents  []Detent
	detent   int
	adaptive bool
	fixed    bool

	// progress is how far shown the sheet is, 0 to 1.
	progress float32
	last     time.Time
	wasOpen  bool

	scrim widget.Clickable
	panel widget.Clickable
	body  *scroll.Scroll

	drag     gesture.Drag
	dragFrom float32
	dragDY   float32
	dragging bool
	panelH   int
}

// New returns a Sheet bound to open: it shows whenever *open is true.
// content builds the body fresh every frame the sheet is shown. The
// styles argument is optional and styles the panel: Background
// (default theme.surface), Padding (default 16 on the sides), Color.
func New(open *bool, content func() elements.Element, st ...styles.Styles) *Sheet {
	s := &Sheet{open: open, content: content, detents: []Detent{DetentFit}}
	if len(st) > 0 {
		s.st = st[0]
	}
	s.body = scroll.New(func() elements.Element { return s.content() }).FitContent(true)
	return s
}

// Title sets the heading shown above the body. It returns s for
// chaining.
func (s *Sheet) Title(t string) *Sheet {
	s.title = t
	return s
}

// Footer sets content pinned below the scrolling body, such as action
// buttons. It returns s for chaining.
func (s *Sheet) Footer(f func() elements.Element) *Sheet {
	s.footer = f
	return s
}

// Detents sets the heights a bottom sheet rests at, smallest first; it
// opens at the first. Dragging the handle moves between them, and
// below the first closes the sheet. It returns s for chaining.
func (s *Sheet) Detents(d ...Detent) *Sheet {
	if len(d) > 0 {
		s.detents = d
		s.detent = 0
	}
	return s
}

// Adaptive shows the sheet as a centered dialog when the window is at
// least 600dp wide, and as a bottom sheet below that. It returns s for
// chaining.
func (s *Sheet) Adaptive(on bool) *Sheet {
	s.adaptive = on
	return s
}

// Dismissible controls whether the scrim, Back, Escape and dragging
// close the sheet (on by default). It returns s for chaining.
func (s *Sheet) Dismissible(on bool) *Sheet {
	s.fixed = !on
	return s
}

// Styles replaces the panel styles from the next frame. It returns s
// for chaining.
func (s *Sheet) Styles(st styles.Styles) *Sheet {
	s.st = st
	return s
}

// Opened reports whether the sheet is open.
func (s *Sheet) Opened() bool {
	return s.open != nil && *s.open
}

func (s *Sheet) close() {
	if s.open != nil && !s.fixed {
		*s.open = false
	}
}

// OutOfFlow marks the sheet as taking no space in a Box, so the box's
// Gap leaves it out; see elements.OutOfFlow.
func (s *Sheet) OutOfFlow() {}
