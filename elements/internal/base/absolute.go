package base

import "image"

// currentOffset accumulates the absolute position of the current
// point in a Layout traversal. Layout is single-threaded and
// synchronous (no concurrent goroutines walk the tree), so a single
// package-level accumulator, pushed before laying out a child and
// popped after, correctly tracks "where am I right now" for anything
// that needs its own absolute screen position - the building block
// behind kit.RegisterPortal (and therefore Tooltip, nested Menu
// submenus and AutoComplete's panel).
var currentOffset image.Point

// AbsolutePosition returns the accumulated offset at the current
// point in the Layout traversal: where the calling element's origin
// currently sits relative to the window. Only meaningful while being
// laid out as part of the current frame.
//
// While a child of a Row or Stack with several children is being laid
// out, it returns the container's origin rather than the child's: a
// flex container measures every child before it knows where any of
// them land. What the child registers with that position - portals
// (RegisterPortal), semantic nodes, positions tracked with
// TrackPosition - is moved to the child's real position once the
// container places it (see ChildTracker), so a dropdown opened from
// the second field of a row still opens below that field. Code that
// stores the position for later should use TrackPosition, which stays
// exact, rather than keep the value returned here.
func AbsolutePosition() image.Point {
	return currentOffset
}

// PushOffset accumulates off into the tracked absolute position;
// pair with the op.Offset(off).Push()/Pop() around the same child. A
// custom element that positions its own children with op.Offset
// should call this alongside it so AbsolutePosition (and therefore
// kit.RegisterPortal) works correctly for content placed inside it.
func PushOffset(off image.Point) (pop func()) {
	currentOffset = currentOffset.Add(off)
	return func() { currentOffset = currentOffset.Sub(off) }
}
