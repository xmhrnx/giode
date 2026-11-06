// Package lazylist provides a vertical list of rows of any height that
// only builds and lays out the rows in or near its viewport - giode's
// FlatList, ListView.builder or LazyColumn.

package lazylist

import (
	"image"

	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles/properties"
)

// Layout lays out the rows in and near the viewport.
func (l *List) Layout(gtx layout.Context) layout.Dimensions {
	viewport := gtx.Constraints.Max
	if viewport.Y >= 1<<24 {
		viewport.Y = max(gtx.Constraints.Min.Y, gtx.Dp(400))
	}
	gtx.Constraints = layout.Exact(viewport)
	l.frame++
	l.viewportH = viewport.Y
	l.n = 0
	if l.count != nil {
		l.n = max(0, l.count())
	}
	if l.width != viewport.X {
		// Heights depend on the width.
		l.width = viewport.X
	}
	if !l.started {
		l.started = true
		l.following = l.anchor == AnchorBottom
	}
	gap := gtx.Dp(l.gap)
	overscan := gtx.Dp(overscanDp)

	// Input: wheel, touch and automation move the anchor.
	l.updateScroll(gtx)
	moved := l.movedBy
	l.movedBy = 0
	l.anchorOff -= moved

	defer clip.Rect{Max: viewport}.Push(gtx.Ops).Pop()
	if bg := properties.CalcColor(l.st.Background); bg.A > 0 {
		paint.Fill(gtx.Ops, bg)
	}
	kit.BlockDragHere(gtx)
	l.scroll.Add(gtx.Ops)

	var rows []placed
	if l.n > 0 {
		rows = l.place(gtx, viewport, gap, overscan)
		// Rows above the anchor, or all of them when following the
		// end, are laid out from the bottom up: a selection area
		// takes their text in screen order.
		tracks := make([]elements.ChildTracker, len(rows))
		for i, p := range rows {
			tracks[i] = p.track
		}
		elements.ReorderTracked(tracks)
	}

	// Paint, and record the visible range.
	popSem := elements.PushSemanticScroll(l)
	l.first, l.last = 0, -1
	for _, p := range rows {
		if p.y+p.h <= 0 || p.y >= viewport.Y {
			continue
		}
		if l.last < l.first {
			l.first = p.idx
		}
		l.last = p.idx
	}
	for _, p := range rows {
		// Rows were laid out before their place was known: move what
		// they registered (semantic nodes, anchors) with them.
		p.track.Place(image.Pt(0, p.y))
		st := op.Offset(image.Pt(0, p.y)).Push(gtx.Ops)
		p.call.Add(gtx.Ops)
		st.Pop()
	}
	popSem()

	// Edge callbacks.
	atTop := l.n > 0 && l.first == 0 && l.last >= 0 && l.anchorIdx == 0 && l.anchorOff >= 0
	if atTop && !l.wasAtTop && l.onReachTop != nil {
		l.onReachTop()
	}
	l.wasAtTop = atTop
	atEnd := l.n > 0 && l.last == l.n-1
	if atEnd && !l.wasAtEnd && l.onReachEnd != nil {
		l.onReachEnd()
	}
	l.wasAtEnd = atEnd

	// The scrollbar, from measured and estimated heights.
	contentH, offset := l.estimate(gap)
	l.contentH = contentH
	l.barOffset = float32(offset)
	before := l.barOffset
	l.bar.Layout(gtx, viewport, contentH, &l.barOffset, l.st.ScrollBar)
	if d := l.barOffset - before; d != 0 {
		l.jumpTo(int(l.barOffset), gap)
	}

	if l.pass == nil {
		l.pass = func(gtx layout.Context) bool {
			return l.updateScroll(gtx) != 0
		}
	}
	kit.HandleInput(l.pass)
	if l.scroll.State() != gesture.StateIdle || moved != 0 {
		kit.MarkLive()
	}
	l.sweep()
	elements.RecordSemantic(gtx, elements.SemanticNode{Kind: elements.SemanticScroll, Target: l}, viewport)
	return layout.Dimensions{Size: viewport}
}

// updateScroll returns the wheel and touch scroll distance since the
// last frame. It stops following the end on a scroll up or when a
// touch drag starts, and follows again when the drag (or its fling)
// ends at the end.
func (l *List) updateScroll(gtx layout.Context) int {
	// The range comes from last frame's estimate; layout clamps the
	// distance to the content's actual ends.
	off := int(l.barOffset)
	rng := pointer.ScrollRange{Min: -off, Max: max(0, l.contentH-l.viewportH-off)}
	if l.following {
		rng.Max = 0
	}
	dist := l.scroll.Update(gtx.Metric, gtx.Source, gtx.Now, gesture.Vertical, pointer.ScrollRange{}, rng)
	if dist < 0 {
		l.following = false
	}
	started, ended := l.gest.Track(&l.scroll, dist)
	switch {
	case started:
		// A finger moves the rows: stop following the end, whichever
		// way it goes, or a growing last row undoes its movement.
		l.following = false
	case ended && l.anchor == AnchorBottom && l.atBottom:
		l.following = true
	}
	l.movedBy += dist
	return dist
}

// place lays out rows around the anchor to fill the viewport and the
// overscan, clamps to the content's ends and updates the anchor.
func (l *List) place(gtx layout.Context, viewport image.Point, gap, overscan int) []placed {
	n := l.n
	idx := l.indexOf(l.anchorKey, l.anchorIdx)
	if idx < 0 {
		idx = min(max(l.anchorIdx, 0), n-1)
	}
	off := l.anchorOff
	if l.hasPending {
		l.hasPending = false
		if p := l.indexOf(l.pendingKey, idx); p >= 0 {
			idx = p
			h := l.heightOf(p)
			switch l.pendingAlign {
			case AlignStart:
				off = 0
			case AlignCenter:
				off = (viewport.Y - h) / 2
			case AlignEnd:
				off = viewport.Y - h
			}
		}
	}
	// A long jump moves the anchor past rows without laying them out,
	// by their measured (or estimated) heights.
	for idx < n-1 && off+l.heightOf(idx)+gap <= -overscan {
		off += l.heightOf(idx) + gap
		idx++
	}
	for idx > 0 && off > viewport.Y+overscan {
		idx--
		off -= l.heightOf(idx) + gap
	}
	var rows []placed
	if l.following {
		// From the end up.
		y := viewport.Y
		for i := n - 1; i >= 0 && y > -overscan; i-- {
			p := l.layoutRow(gtx, i)
			y -= p.h
			p.y = y
			rows = append(rows, p)
			y -= gap
		}
		reverse(rows)
		l.atBottom = true
		if top := rows[0]; top.idx == 0 && top.y > 0 {
			// Shorter than the viewport: from the top.
			shift := -top.y
			for i := range rows {
				rows[i].y += shift
			}
		}
	} else {
		// From the anchor down, then up to fill any space above.
		y := off
		for i := idx; i < n && y < viewport.Y+overscan; i++ {
			p := l.layoutRow(gtx, i)
			p.y = y
			rows = append(rows, p)
			y += p.h + gap
		}
		// Past the end: pull the rows down so the last one ends at the
		// bottom.
		if last := rows[len(rows)-1]; last.idx == n-1 && last.y+last.h < viewport.Y {
			shift := viewport.Y - (last.y + last.h)
			for i := range rows {
				rows[i].y += shift
			}
		}
		y = rows[0].y
		var above []placed
		for i := rows[0].idx - 1; i >= 0 && y > -overscan; i-- {
			p := l.layoutRow(gtx, i)
			y -= gap + p.h
			p.y = y
			above = append(above, p)
		}
		reverse(above)
		rows = append(above, rows...)
		// Above the start: the first row starts at the top.
		if top := rows[0]; top.idx == 0 && top.y > 0 {
			shift := -top.y
			for i := range rows {
				rows[i].y += shift
			}
		}
		last := rows[len(rows)-1]
		l.atBottom = last.idx == n-1 && last.y+last.h <= viewport.Y
		if l.anchor == AnchorBottom && !l.gest.Moving() {
			// Back at the end: follow it again - but not while a finger
			// is still moving the rows.
			l.following = l.atBottom
		}
	}
	// The anchor: the first row whose bottom is in view.
	for _, p := range rows {
		if p.y+p.h > 0 || p.idx == rows[len(rows)-1].idx {
			l.anchorIdx, l.anchorKey, l.anchorOff = p.idx, l.key(p.idx), p.y
			break
		}
	}
	return rows
}

// estimate returns the content height and the offset of the viewport
// into it, from measured heights and the average for unmeasured rows.
func (l *List) estimate(gap int) (contentH, offset int) {
	if l.n == 0 {
		return 0, 0
	}
	// Rows are summed only up to a limit: past it, the average stands
	// in, keeping long lists cheap.
	const exact = 2000
	sum := func(from, to int) int {
		if to-from > exact {
			return (to - from) * (max(1, l.measure) + gap)
		}
		s := 0
		for i := from; i < to; i++ {
			s += l.heightOf(i) + gap
		}
		return s
	}
	e := &l.est
	if !e.valid || e.n != l.n || e.anchorIdx != l.anchorIdx || e.gap != gap || e.measure != l.measure || l.frame-e.frame >= estimateFrames {
		e.above, e.below = sum(0, l.anchorIdx), sum(l.anchorIdx, l.n)
		e.valid, e.n, e.anchorIdx, e.gap, e.measure, e.frame = true, l.n, l.anchorIdx, gap, l.measure, l.frame
	}
	return e.above + e.below - gap, e.above - l.anchorOff
}

// estimateFrames is how often estimate sums the rows' heights anew when
// nothing it knows of changed: a row edited offscreen gets a new
// version, and so its height the estimate, which only the scrollbar
// shows.
const estimateFrames = 30

// jumpTo moves the anchor to the row at the estimated offset y, for a
// scrollbar drag.
func (l *List) jumpTo(y, gap int) {
	acc := 0
	for i := 0; i < l.n; i++ {
		h := l.heightOf(i) + gap
		if acc+h > y || i == l.n-1 {
			l.anchorIdx, l.anchorKey, l.anchorOff = i, l.key(i), acc-y
			l.following = false
			return
		}
		acc += h
	}
}
