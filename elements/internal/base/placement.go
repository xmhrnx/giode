package base

import "image"

// ChildTracker records what a child registers while it's laid out -
// portals, semantic nodes and tracked positions, all placed at the
// absolute position current when it starts - so they can be moved to
// where the child actually ends up. Containers that lay a child out
// before they know its final offset use it (giode's Row, Stack and
// Scroll do):
//
//	t := elements.TrackChild()
//	dims := child.Layout(gtx) // recorded, offset not yet known
//	t.End()
//	... // decide where the child goes
//	t.Place(off)            // shift what it registered by off
//	defer op.Offset(off).Push(gtx.Ops).Pop()
//	call.Add(gtx.Ops)
type ChildTracker struct {
	semStart, semEnd       int
	portalStart, portalEnd int
	probeStart, probeEnd   int
	// selStart and selEnd bound the text the child registered with the
	// innermost selection area; -1 outside one.
	selStart, selEnd int
}

// probes are the positions registered with TrackPosition this frame,
// shifted along with everything else when an enclosing container
// places the child that registered them.
var probes []*image.Point

// TrackPosition stores the calling element's absolute position in *p
// and keeps it exact: when an enclosing Row, Stack or Scroll learns
// where it really put the element - which a flex container only knows
// after laying every child out - *p is shifted to match. Read *p once
// the frame's layout is done: in the next frame's event handling, a
// click handler, or a portal's build function.
//
//	prev := t.origin // exact, from the last frame
//	elements.TrackPosition(&t.origin)
//
// A measuring pass (Measure) leaves *p alone: its offset isn't where the
// element goes, and nothing corrects it afterwards, so a measure after
// the real layout would leave *p wrong.
func TrackPosition(p *image.Point) {
	if measuring > 0 {
		return
	}
	*p = currentOffset
	probes = append(probes, p)
}

// TrackChild starts recording; call it right before laying the child
// out.
func TrackChild() ChildTracker {
	t := ChildTracker{portalStart: len(portalQueue), probeStart: len(probes)}
	if semantics != nil {
		t.semStart = len(*semantics)
	}
	t.semEnd, t.portalEnd, t.probeEnd = t.semStart, t.portalStart, t.probeStart
	t.selStart = SelectionLen()
	t.selEnd = t.selStart
	return t
}

// End stops recording; call it right after the child's Layout returns.
func (t *ChildTracker) End() {
	t.portalEnd, t.probeEnd = len(portalQueue), len(probes)
	if semantics != nil {
		t.semEnd = len(*semantics)
	}
	t.selEnd = SelectionLen()
}

// SelectionLen is how many text parts the innermost selection area
// holds so far this frame, -1 outside one; SelectionReorder rearranges
// them: spans, ranges of parts, in the order they're to take. The text
// package sets both.
var (
	SelectionLen     = func() int { return -1 }
	SelectionReorder = func(spans [][2]int) {}
)

// ReorderTracked tells the innermost selection area the screen order of
// children a container laid out in another order - a list laying its
// rows out from the bottom up - so selecting by dragging runs through
// them top to bottom. ts are the children's trackers, in screen order.
func ReorderTracked(ts []ChildTracker) {
	if len(ts) < 2 {
		return
	}
	lo, hi, total := -1, -1, 0
	spans := make([][2]int, 0, len(ts))
	inOrder := true
	for i, t := range ts {
		if t.selStart < 0 || t.selEnd < t.selStart {
			return
		}
		if i > 0 && t.selStart < ts[i-1].selEnd {
			inOrder = false
		}
		if lo < 0 || t.selStart < lo {
			lo = t.selStart
		}
		hi = max(hi, t.selEnd)
		total += t.selEnd - t.selStart
		spans = append(spans, [2]int{t.selStart, t.selEnd})
	}
	if inOrder || total != hi-lo {
		// Already in order, or not the contiguous run of parts this
		// expects: leave them.
		return
	}
	SelectionReorder(spans)
}

// Place moves everything recorded by off, the child's offset from the
// position it was laid out at.
func (t ChildTracker) Place(off image.Point) {
	if off == (image.Point{}) {
		return
	}
	if semantics != nil {
		nodes := *semantics
		for i := t.semStart; i < t.semEnd && i < len(nodes); i++ {
			nodes[i].Bounds = nodes[i].Bounds.Add(off)
		}
	}
	for i := t.portalStart; i < t.portalEnd && i < len(portalQueue); i++ {
		if !portalQueue[i].overlay && !portalQueue[i].fixed {
			portalQueue[i].anchor = portalQueue[i].anchor.Add(off)
		}
	}
	for i := t.probeStart; i < t.probeEnd && i < len(probes); i++ {
		*probes[i] = probes[i].Add(off)
	}
}
