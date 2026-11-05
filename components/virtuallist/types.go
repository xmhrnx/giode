// Package virtuallist provides a vertical list that only builds the
// rows in or near its viewport, so it handles any number of items at a
// constant cost per frame.

package virtuallist

// Align places an item in the viewport for ScrollToIndex.
type Align int

const (
	// AlignNearest scrolls as little as possible to show the item
	// fully: not at all if it already is.
	AlignNearest Align = iota
	// AlignStart puts the item at the top.
	AlignStart
	// AlignCenter centers the item.
	AlignCenter
	// AlignEnd puts the item at the bottom.
	AlignEnd
)

const defaultOverscan = 3

var defaultFocus = "theme.primary/0.18"
