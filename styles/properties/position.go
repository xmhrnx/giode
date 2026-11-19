package properties

// Position selects how an element is placed (CSS position).
type Position string

const (
	// PositionStatic is the default: the element takes part in the
	// normal flow.
	PositionStatic Position = "static"
	// PositionRelative behaves like static but can act as the
	// containing block of absolute descendants.
	PositionRelative Position = "relative"
	// PositionAbsolute removes the element from the flow and places
	// it at the Top/Right/Bottom/Left offsets of its parent box,
	// painting above the flow content.
	PositionAbsolute Position = "absolute"
	// PositionSticky keeps its place in the flow, but when the
	// enclosing Scroll scrolls it past the top, it stays at the top
	// (Top dp below it) until the next sticky element pushes it out:
	// table headers, a chat's day headers, section titles.
	PositionSticky Position = "sticky"
	// PositionStickyStart sticks at the start edge (left, or right
	// right to left; Left dp in) of a Scroll scrolling sideways: a
	// table's frozen first column.
	PositionStickyStart Position = "sticky-start"
	// PositionStickyTopStart sticks at both: a table's corner cell.
	PositionStickyTopStart Position = "sticky-top-start"
)

// StickyTop and StickyStart report the edges a sticky position keeps
// to.
func (p Position) StickyTop() bool {
	return p == PositionSticky || p == PositionStickyTopStart
}

func (p Position) StickyStart() bool {
	return p == PositionStickyStart || p == PositionStickyTopStart
}

// Mode resolves the value, defaulting to static.
func (p Position) Mode() Position {
	if p == "" {
		return PositionStatic
	}
	return p
}
