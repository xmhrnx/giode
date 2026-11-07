// Package tabs provides a stateful tab bar with per-tab views.
package tabs

import (
	"gioui.org/widget"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
)

var (
	defaultActive     = "#3b82f6"
	defaultText       = "#94a3b8"
	defaultActiveText = "#f8fafc"
)

// Tab is one tab: its label and the view shown while selected.
type Tab struct {
	Label string
	View  func() elements.Element
}

// Tabs is a stateful tab bar. Create it once with New and lay it out
// every frame; only the selected tab's view is built.
type Tabs struct {
	// Contextual gives it a context menu (Context).
	elements.Contextual

	tabs     []Tab
	selected int
	clicks   []*widget.Clickable
	onChange func(index int)
	st       styles.Styles
}

// New returns a Tabs with the given styles and tabs: Color is the
// active indicator and BorderColor is the active text.
func New(st styles.Styles, tabs ...Tab) *Tabs {
	t := &Tabs{tabs: tabs, st: st}
	for range tabs {
		t.clicks = append(t.clicks, new(widget.Clickable))
	}
	return t
}

// Selected returns the index of the active tab.
func (t *Tabs) Selected() int {
	return t.selected
}

// SetSelected switches the active tab without invoking OnChange.
func (t *Tabs) SetSelected(i int) {
	if i >= 0 && i < len(t.tabs) {
		t.selected = i
	}
}

// OnChange sets the handler invoked when the active tab changes. It
// returns t for chaining.
func (t *Tabs) OnChange(fn func(index int)) *Tabs {
	t.onChange = fn
	return t
}
