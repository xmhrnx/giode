package box

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/elements/internal/base"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// gridItem is a child placed on the grid: columns [c0, c1), rows
// [r0, r1).
type gridItem struct {
	el             base.Element
	c0, c1, r0, r1 int
	// h is its height measured at its columns' width.
	h int
}

// layoutGrid lays children out on the grid st's GridColumns and
// GridRows describe (Display: DisplayGrid), after CSS grid.
func (b *BoxElement) layoutGrid(gtx layout.Context, st styles.Styles, children []base.Element, zs *[]zPaint) layout.Dimensions {
	colGap := int(st.Gap)
	rowGap := colGap
	if st.RowGap != 0 {
		rowGap = int(st.RowGap)
	}
	maxW := gtx.Constraints.Max.X
	bounded := !base.IsUnbounded(maxW)
	dp := func(v float32) int { return gtx.Dp(unit.Dp(v)) }

	// Columns: the spec, auto-fill expanded for the width there is.
	spec := st.GridColumns.Parse()
	cols := spec.Tracks
	if spec.AutoFill {
		t := cols[0]
		least := max(dp(max(t.Min, t.Size)), 1)
		if t.Kind == properties.TrackFr && t.Min == 0 {
			least = 1
		}
		n := 1
		if bounded {
			n = max((maxW+colGap)/(least+colGap), 1)
		}
		cols = make([]properties.Track, n)
		for i := range cols {
			cols[i] = t
		}
	}
	if len(cols) == 0 {
		cols = []properties.Track{{Kind: properties.TrackFr, Size: 1}}
	}

	// Placement: explicit positions first, then the rest in reading
	// order, each in the first free cells from where the last went.
	items := make([]gridItem, len(children))
	ncols := len(cols)
	occupied := map[[2]int]bool{}
	fits := func(r, c, cs, rs int) bool {
		if c+cs > ncols {
			return false
		}
		for y := r; y < r+rs; y++ {
			for x := c; x < c+cs; x++ {
				if occupied[[2]int{y, x}] {
					return false
				}
			}
		}
		return true
	}
	take := func(it *gridItem) {
		for y := it.r0; y < it.r1; y++ {
			for x := it.c0; x < it.c1; x++ {
				occupied[[2]int{y, x}] = true
			}
		}
	}
	span := func(s properties.GridSpan) int { return max(s.Span, 1) }
	placed := make([]bool, len(children))
	for i, child := range children {
		c, r := base.GridPlacementOf(child)
		items[i].el = child
		if c.Start > 0 && r.Start > 0 {
			cs := min(span(c), ncols)
			c0 := min(c.Start-1, ncols-cs)
			items[i].c0, items[i].c1 = c0, c0+cs
			items[i].r0, items[i].r1 = r.Start-1, r.Start-1+span(r)
			take(&items[i])
			placed[i] = true
		}
	}
	row, col := 0, 0
	for i, child := range children {
		if placed[i] {
			continue
		}
		c, r := base.GridPlacementOf(child)
		cs, rs := min(span(c), ncols), span(r)
		switch {
		case c.Start > 0:
			// A fixed column: the first free row from the cursor's.
			c0 := min(c.Start-1, ncols-cs)
			y := row
			for !fits(y, c0, cs, rs) {
				y++
			}
			items[i].c0, items[i].c1, items[i].r0, items[i].r1 = c0, c0+cs, y, y+rs
		case r.Start > 0:
			y := r.Start - 1
			x := 0
			for !fits(y, x, cs, rs) {
				x++
				if x+cs > ncols {
					x = 0
					y++
				}
			}
			items[i].c0, items[i].c1, items[i].r0, items[i].r1 = x, x+cs, y, y+rs
		default:
			for !fits(row, col, cs, rs) {
				col++
				if col+cs > ncols {
					col = 0
					row++
				}
			}
			items[i].c0, items[i].c1, items[i].r0, items[i].r1 = col, col+cs, row, row+rs
			col += cs
			if col >= ncols {
				col = 0
				row++
			}
		}
		take(&items[i])
	}

	// Column widths: fixed, then auto at their content's width, then
	// the fractions share what's left, none below its minimum.
	widths := make([]int, ncols)
	gaps := colGap * (ncols - 1)
	frTotal := float32(0)
	for i, t := range cols {
		switch t.Kind {
		case properties.TrackFixed:
			widths[i] = dp(t.Size)
		case properties.TrackFr:
			frTotal += t.Size
			widths[i] = dp(t.Min)
		}
	}
	for ci, t := range cols {
		if t.Kind != properties.TrackAuto {
			continue
		}
		w := dp(t.Min)
		for _, it := range items {
			if it.c0 == ci && it.c1 == ci+1 {
				mg := gtx
				mg.Constraints = layout.Constraints{Max: image.Pt(maxW, base.Unbounded)}
				w = max(w, base.Measure(mg, it.el).Size.X)
			}
		}
		widths[ci] = w
	}
	if frTotal > 0 {
		if bounded {
			// The fractions share the rest; one whose share is below its
			// minimum keeps the minimum and leaves the rest to the others.
			fixed := gaps
			for i, t := range cols {
				if t.Kind != properties.TrackFr {
					fixed += widths[i]
				}
			}
			frozen := make([]bool, ncols)
			for {
				left, total := maxW-fixed, float32(0)
				for i, t := range cols {
					if t.Kind == properties.TrackFr && !frozen[i] {
						total += t.Size
					} else if t.Kind == properties.TrackFr {
						left -= widths[i]
					}
				}
				changed := false
				for i, t := range cols {
					if t.Kind != properties.TrackFr || frozen[i] {
						continue
					}
					share := int(float32(max(left, 0)) * t.Size / total)
					if share < dp(t.Min) {
						widths[i], frozen[i], changed = dp(t.Min), true, true
					} else {
						widths[i] = share
					}
				}
				if !changed {
					break
				}
			}
		} else {
			// No width to share: fractions size to their content.
			for ci, t := range cols {
				if t.Kind != properties.TrackFr {
					continue
				}
				for _, it := range items {
					if it.c0 == ci && it.c1 == ci+1 {
						mg := gtx
						mg.Constraints = layout.Constraints{Max: image.Pt(base.Unbounded, base.Unbounded)}
						widths[ci] = max(widths[ci], base.Measure(mg, it.el).Size.X)
					}
				}
			}
		}
	}
	colX := make([]int, ncols+1)
	for i := range ncols {
		colX[i+1] = colX[i] + widths[i] + colGap
	}
	spanW := func(it gridItem) int { return colX[it.c1] - colX[it.c0] - colGap }

	// Rows: fixed ones as given, auto ones as tall as their tallest
	// one-row item; items spanning rows grow the last of them.
	nrows := 0
	for _, it := range items {
		nrows = max(nrows, it.r1)
	}
	rspec := st.GridRows.Parse().Tracks
	heights := make([]int, nrows)
	rowTrack := func(r int) properties.Track {
		if r < len(rspec) {
			return rspec[r]
		}
		return properties.Track{}
	}
	for i := range items {
		it := &items[i]
		mg := gtx
		mg.Constraints = layout.Constraints{Min: image.Pt(spanW(*it), 0), Max: image.Pt(spanW(*it), base.Unbounded)}
		it.h = layoutStretched(it.el, true, false, func() layout.Dimensions { return base.Measure(mg, it.el) }).Size.Y
	}
	for r := range nrows {
		t := rowTrack(r)
		if t.Kind == properties.TrackFixed {
			heights[r] = dp(t.Size)
			continue
		}
		heights[r] = dp(t.Min)
		for _, it := range items {
			if it.r0 == r && it.r1 == r+1 {
				heights[r] = max(heights[r], it.h)
			}
		}
	}
	for _, it := range items {
		if it.r1-it.r0 < 2 {
			continue
		}
		have := rowGap * (it.r1 - it.r0 - 1)
		for r := it.r0; r < it.r1; r++ {
			have += heights[r]
		}
		if it.h > have && rowTrack(it.r1-1).Kind != properties.TrackFixed {
			heights[it.r1-1] += it.h - have
		}
	}
	// Fractional rows share a fixed height's leftover.
	if minH := gtx.Constraints.Min.Y; minH > 0 {
		used, frs := rowGap*max(nrows-1, 0), float32(0)
		for r := range nrows {
			used += heights[r]
			if t := rowTrack(r); t.Kind == properties.TrackFr {
				frs += t.Size
			}
		}
		if extra := minH - used; extra > 0 && frs > 0 {
			for r := range nrows {
				if t := rowTrack(r); t.Kind == properties.TrackFr {
					heights[r] += int(float32(extra) * t.Size / frs)
				}
			}
		}
	}
	rowY := make([]int, nrows+1)
	for i := range nrows {
		rowY[i+1] = rowY[i] + heights[i] + rowGap
	}

	// Each item laid out in its cells: as wide as its columns, and as
	// tall as its rows unless aligned to the start, center or end.
	for _, it := range items {
		w := spanW(it)
		h := rowY[it.r1] - rowY[it.r0] - rowGap
		cgtx := gtx
		cgtx.Constraints = layout.Constraints{Min: image.Pt(w, 0), Max: image.Pt(w, h)}
		stretch := st.Align == "" || st.Align == properties.AlignStretch || st.Align == properties.AlignStart
		if st.Align == "" || st.Align == properties.AlignStretch {
			cgtx.Constraints.Min.Y = h
		}
		track := base.TrackChild()
		macro := op.Record(gtx.Ops)
		dims := layoutStretched(it.el, true, cgtx.Constraints.Min.Y > 0, func() layout.Dimensions { return it.el.Layout(cgtx) })
		ops := macro.Stop()
		track.End()
		y := rowY[it.r0]
		if !stretch {
			y += alignOffset(st.Align, h, dims.Size.Y)
		}
		pos := image.Pt(colX[it.c0], y)
		track.Place(pos)
		if zs != nil {
			*zs = append(*zs, zPaint{call: ops, off: pos, size: image.Pt(w, dims.Size.Y), z: base.ZIndexOf(it.el), track: track})
		} else {
			off := op.Offset(pos).Push(gtx.Ops)
			ops.Add(gtx.Ops)
			off.Pop()
		}
	}

	size := image.Pt(max(colX[ncols]-colGap, 0), max(rowY[nrows]-rowGap, 0))
	if frTotal > 0 && bounded {
		size.X = maxW
	}
	return layout.Dimensions{Size: gtx.Constraints.Constrain(size)}
}

// layoutStretched runs layout with a Box child filling its cell's
// width, and with y its height, as a Box sized by its content wouldn't
// (see stretchChild).
func layoutStretched(el base.Element, x, y bool, layout func() layout.Dimensions) layout.Dimensions {
	if b, ok := el.(*BoxElement); ok {
		sx, sy := b.stretchX, b.stretchY
		b.stretchX, b.stretchY = x || sx, y || sy
		defer func() { b.stretchX, b.stretchY = sx, sy }()
	}
	return layout()
}
