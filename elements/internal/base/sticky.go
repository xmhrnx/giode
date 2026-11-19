package base

import (
	"image"

	"gioui.org/op"
)

// Sticky elements (PositionSticky and the like) keep their place in
// the flow, but the enclosing Scroll paints them: only it knows, once
// its content is placed and scrolled, where they show. A sticky
// element lays itself out into a recording and adds a StickyItem to the
// innermost StickyScope instead of painting; the Scroll paints the
// items after its content, held at its edges.

// StickyItem is a sticky element waiting to be painted.
type StickyItem struct {
	// Call paints it; Size is its size.
	Call op.CallOp
	Size image.Point
	// At is its position in the flow, absolute, kept exact as its
	// containers place it (TrackPosition).
	At image.Point
	// Top and Start stick it to those edges, Top and Left px in.
	Top, Start           bool
	TopOffset, StartOffs int
	// Track holds what it registered (portals, semantic nodes), moved
	// to where it's painted.
	Track ChildTracker
}

// StickyScope collects the sticky elements of a scroll's content.
type StickyScope struct {
	Items []*StickyItem
}

var stickyStack []*StickyScope

// stickyCount counts the sticky elements handed to scopes: a memo
// holding one lays its subtree out every frame (see MemoCache.record).
var stickyCount int

// PushStickyScope makes s collect the sticky elements laid out until
// pop runs. A measure (Measure) collects none.
func PushStickyScope(s *StickyScope) (pop func()) {
	stickyStack = append(stickyStack, s)
	n := len(stickyStack)
	return func() { stickyStack = stickyStack[:n-1] }
}

// AddSticky hands it to the innermost scope, and reports whether there
// was one: without one, the element paints itself in the flow.
func AddSticky(it *StickyItem) bool {
	if len(stickyStack) == 0 || measuring > 0 {
		return false
	}
	s := stickyStack[len(stickyStack)-1]
	s.Items = append(s.Items, it)
	stickyCount++
	return true
}
