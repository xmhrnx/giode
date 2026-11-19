package base

import "image"

// Drag and drop support shared with the dnd package: whether a drag is
// in progress, which cached subtrees take part in one, and the visible
// area of scrolling containers.

var (
	// dragActive is set while an in-app drag is in progress.
	dragActive bool
	// dragEpoch changes when a drag starts or ends.
	dragEpoch uint64
	// dragAware counts the drag sources, drop targets and list slots
	// laid out: a memo holding one lays its subtree out while a drag is
	// in progress (see MemoCache.record), as targets register their
	// places every frame.
	dragAware int
)

// SetDragActive records that a drag started (true) or ended (false).
// The dnd package calls it.
func SetDragActive(v bool) {
	if dragActive != v {
		dragActive = v
		dragEpoch++
	}
}

// DragActive reports whether a drag is in progress.
func DragActive() bool { return dragActive }

// NoteDragAware reports that the element being laid out takes part in
// drag and drop.
func NoteDragAware() {
	if measuring == 0 {
		dragAware++
	}
}

// VisibleClip is the part of the window a scrolling container shows:
// content it lays out is visible only inside it, and inside each
// enclosing one. Drop targets use it so a target scrolled partly out of
// view only takes drops where it shows.
type VisibleClip struct {
	// Origin is the container's window position, kept exact
	// (TrackPosition); Size its viewport.
	Origin image.Point
	Size   image.Point
	// Parent is the enclosing clip, nil at the window.
	Parent *VisibleClip
}

// Rect is the visible area: the clip's rectangle within its parents'.
func (c *VisibleClip) Rect() image.Rectangle {
	r := image.Rectangle{Min: c.Origin, Max: c.Origin.Add(c.Size)}
	for p := c.Parent; p != nil; p = p.Parent {
		r = r.Intersect(image.Rectangle{Min: p.Origin, Max: p.Origin.Add(p.Size)})
	}
	return r
}

var clipStack []*VisibleClip

// PushVisibleClip makes c the visible area of the content laid out
// until pop runs. Call it at the container's own position, before
// laying out (and tracking) its scrolled content, and set c.Size to the
// viewport before the frame ends.
func PushVisibleClip(c *VisibleClip) (pop func()) {
	c.Parent = CurrentVisibleClip()
	TrackPosition(&c.Origin)
	clipStack = append(clipStack, c)
	n := len(clipStack)
	return func() { clipStack = clipStack[:n-1] }
}

// CurrentVisibleClip is the innermost visible clip, nil outside any
// scrolling container.
func CurrentVisibleClip() *VisibleClip {
	if len(clipStack) == 0 {
		return nil
	}
	return clipStack[len(clipStack)-1]
}
