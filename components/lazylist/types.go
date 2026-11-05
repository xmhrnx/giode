// Package lazylist provides a vertical list of rows of any height that
// only builds and lays out the rows in or near its viewport - giode's
// FlatList, ListView.builder or LazyColumn.

package lazylist

// Anchor selects where the list holds on when content changes.
type Anchor int

const (
	// AnchorTop keeps the rows in view where they are as rows are
	// added, removed or resized (the default).
	AnchorTop Anchor = iota
	// AnchorBottom starts at the end and keeps to it as rows are added
	// or the last row grows - a chat - until the user scrolls up; it
	// follows again once they're back at the end.
	AnchorBottom
)

// Align places a row in the viewport for ScrollToKey.
type Align int

const (
	// AlignStart puts the row's top at the top of the viewport.
	AlignStart Align = iota
	// AlignCenter centers the row.
	AlignCenter
	// AlignEnd puts the row's bottom at the bottom of the viewport.
	AlignEnd
)

// overscanDp is how far past the viewport rows are laid out, so a
// scroll doesn't show them appear.
const overscanDp = 300
