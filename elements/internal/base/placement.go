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
func TrackPosition(p *image.Point) {
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
	return t
}

// End stops recording; call it right after the child's Layout returns.
func (t *ChildTracker) End() {
	t.portalEnd, t.probeEnd = len(portalQueue), len(probes)
	if semantics != nil {
		t.semEnd = len(*semantics)
	}
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
