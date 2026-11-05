// Package scroll provides a stateful vertical scroll container.

package scroll

const (
	// unbounded mirrors the sentinel Gio uses for infinite
	// constraints.
	unbounded = 1e6
)

// Axis selects the directions a Scroll scrolls in.
type Axis int

const (
	// Vertical scrolls up and down (the default).
	Vertical Axis = 1 << iota
	// Horizontal scrolls sideways, for wide code or tables.
	Horizontal
)

// Anchor selects where a Scroll keeps its content when it grows.
type Anchor int

const (
	// AnchorTop keeps the offset when content is added (the default).
	AnchorTop Anchor = iota
	// AnchorBottom keeps a scroll that's at the bottom there as its
	// content grows - a chat following a reply as it streams in - and
	// stops following once the user scrolls up, until they scroll
	// back to the bottom. It starts at the bottom.
	AnchorBottom
)
