package base

import (
	"image"
	"sort"

	"gioui.org/layout"
	"gioui.org/op"
)

// PaintSticky paints the content's sticky elements over it: each where
// the scrolled flow put it, or held at the top (or start) edge once
// scrolled past it, until the next one sticking to that edge pushes it
// out, as section headers do. origin is the scroll's absolute position;
// the items' positions are absolute, scrolled.
func PaintSticky(gtx layout.Context, items []*StickyItem, origin, viewport image.Point, rtl bool) {
	if len(items) == 0 {
		return
	}
	type placed struct {
		it   *StickyItem
		flow image.Point // in the viewport, as scrolled
		at   image.Point
	}
	ps := make([]placed, len(items))
	for i, it := range items {
		p := it.At.Sub(origin)
		ps[i] = placed{it: it, flow: p, at: p}
	}
	// Top: held at the top, and pushed up by the next item held there.
	sort.SliceStable(ps, func(i, j int) bool { return ps[i].flow.Y < ps[j].flow.Y })
	for i := range ps {
		it := ps[i].it
		if !it.Top {
			continue
		}
		y := max(ps[i].flow.Y, it.TopOffset)
		for j := i + 1; j < len(ps); j++ {
			// Items on the same line (a header row's cells) stay side by
			// side; one further down pushes.
			if ps[j].it.Top && ps[j].flow.Y > ps[i].flow.Y {
				y = min(y, ps[j].flow.Y-it.Size.Y)
				break
			}
		}
		ps[i].at.Y = max(y, ps[i].flow.Y)
	}
	// Start: held at the left edge (the right one right to left).
	for i := range ps {
		it := ps[i].it
		if !it.Start {
			continue
		}
		if rtl {
			ps[i].at.X = min(ps[i].flow.X, viewport.X-it.Size.X-it.StartOffs)
		} else {
			ps[i].at.X = max(ps[i].flow.X, it.StartOffs)
		}
	}
	// Items held at both edges (corners) go last, over the others.
	rank := func(it *StickyItem) int {
		switch {
		case it.Top && it.Start:
			return 2
		case it.Start:
			return 1
		}
		return 0
	}
	sort.SliceStable(ps, func(i, j int) bool { return rank(ps[i].it) < rank(ps[j].it) })
	for _, p := range ps {
		delta := p.at.Sub(p.flow)
		p.it.Track.Place(delta)
		off := op.Offset(p.at).Push(gtx.Ops)
		p.it.Call.Add(gtx.Ops)
		off.Pop()
	}
}
