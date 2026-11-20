package richedit

import (
	"image"

	"github.com/xmhrnx/giode/internal/textrender"
	"github.com/xmhrnx/giode/richdoc"
)

// Coordinates here are the content's: x from the editor's left, y from
// the document's top (the scroll offset not applied).

// textOrigin is where block i's text starts.
func (e *RichEditor) textOrigin(i int, v *blockView) image.Point {
	return image.Pt(v.g.indent, e.tops[i]+v.g.padTop)
}

// stopFor returns the line and stop of a caret at rune r: the last such
// stop, so a caret at a wrap or after a newline is at the next line's
// start.
func stopFor(lines []textrender.StopLine, r int) (li int, s textrender.Stop) {
	li, best := 0, -1
	for i, ln := range lines {
		for _, st := range ln.Stops {
			d := st.Rune - r
			if d < 0 {
				d = -d
			}
			if best < 0 || d < best || d == best && i > li {
				li, s, best = i, st, d
			}
		}
	}
	return li, s
}

// caretRect is the caret's box at p.
func (e *RichEditor) caretRect(p richdoc.Pos) (image.Rectangle, bool) {
	if p.Block >= len(e.laid) || e.laid[p.Block] == nil {
		return image.Rectangle{}, false
	}
	v := e.laid[p.Block]
	b := &e.st.Doc.Blocks[p.Block]
	top := e.tops[p.Block]
	if !b.Kind.HasText() {
		x := 0
		if p.Offset > 0 {
			x = e.width
			if b.Kind == richdoc.Image {
				x = min(e.width, v.img.X)
			}
		}
		return image.Rect(x-1, top, x+1, top+v.height-v.g.padBottom), true
	}
	if len(v.lines) == 0 {
		o := e.textOrigin(p.Block, v)
		return image.Rect(o.X-1, o.Y, o.X+1, o.Y+v.laid.Dims.Size.Y), true
	}
	li, s := stopFor(v.lines, p.Offset)
	o := e.textOrigin(p.Block, v)
	ln := v.lines[li]
	return image.Rect(o.X+s.X-1, o.Y+ln.Top, o.X+s.X+1, o.Y+ln.Bottom), true
}

// blockAt returns the block at content y.
func (e *RichEditor) blockAt(y int) int {
	n := len(e.tops)
	lo, hi := 0, n-1
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if e.tops[mid] <= y {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return max(lo, 0)
}

// hit returns the position nearest content point p.
func (e *RichEditor) hit(p image.Point) richdoc.Pos {
	if len(e.tops) == 0 {
		return richdoc.Pos{}
	}
	i := e.blockAt(p.Y)
	if i >= len(e.laid) || e.laid[i] == nil {
		return richdoc.Pos{Block: i}
	}
	v := e.laid[i]
	b := &e.st.Doc.Blocks[i]
	if !b.Kind.HasText() {
		off := 0
		if p.X > e.width/2 || b.Kind == richdoc.Image && p.X > v.img.X/2 {
			off = 1
		}
		return richdoc.Pos{Block: i, Offset: off}
	}
	o := e.textOrigin(i, v)
	if len(v.lines) == 0 {
		return richdoc.Pos{Block: i}
	}
	r := textrender.HitRune(v.lines, p.Sub(o))
	return richdoc.Pos{Block: i, Offset: min(max(r, 0), b.Len())}
}

// lineOf returns the index of p's line in its block, and how many there
// are.
func (e *RichEditor) lineOf(p richdoc.Pos) (int, int) {
	if p.Block >= len(e.laid) {
		return 0, 1
	}
	v := e.laid[p.Block]
	if v == nil || len(v.lines) == 0 {
		return 0, 1
	}
	li, _ := stopFor(v.lines, p.Offset)
	return li, len(v.lines)
}

// vertical moves p a line up (dir < 0) or down, keeping x; at the first
// or last line it goes to the start or end. ok is false when it can't
// move: the needed block isn't laid out.
func (e *RichEditor) vertical(p richdoc.Pos, dir, x int) (richdoc.Pos, bool) {
	if p.Block >= len(e.laid) || e.laid[p.Block] == nil {
		return p, false
	}
	r, _ := e.caretRect(p)
	li, n := e.lineOf(p)
	target := 0
	switch {
	case dir < 0 && (li > 0 && e.st.Doc.Blocks[p.Block].Kind.HasText()):
		ln := e.laid[p.Block].lines[li-1]
		target = e.tops[p.Block] + e.laid[p.Block].g.padTop + (ln.Top+ln.Bottom)/2
	case dir > 0 && li < n-1 && e.st.Doc.Blocks[p.Block].Kind.HasText():
		ln := e.laid[p.Block].lines[li+1]
		target = e.tops[p.Block] + e.laid[p.Block].g.padTop + (ln.Top+ln.Bottom)/2
	case dir < 0:
		if p.Block == 0 {
			return richdoc.Pos{}, true
		}
		target = r.Min.Y - e.laid[p.Block].g.padTop - 1
		prev := p.Block - 1
		if prev >= len(e.laid) || e.laid[prev] == nil {
			return p, false
		}
		pv := e.laid[prev]
		if len(pv.lines) > 0 {
			ln := pv.lines[len(pv.lines)-1]
			target = e.tops[prev] + pv.g.padTop + (ln.Top+ln.Bottom)/2
		} else {
			target = e.tops[prev] + pv.height/2
		}
	default:
		if p.Block == len(e.st.Doc.Blocks)-1 {
			return e.st.Doc.End(), true
		}
		next := p.Block + 1
		if next >= len(e.laid) || e.laid[next] == nil {
			return p, false
		}
		nv := e.laid[next]
		if len(nv.lines) > 0 {
			ln := nv.lines[0]
			target = e.tops[next] + nv.g.padTop + (ln.Top+ln.Bottom)/2
		} else {
			target = e.tops[next] + nv.height/2
		}
	}
	return e.hit(image.Pt(x, target)), true
}

// lineEdge returns the start (or end) of p's line.
func (e *RichEditor) lineEdge(p richdoc.Pos, end bool) richdoc.Pos {
	var v *blockView
	if p.Block < len(e.laid) {
		v = e.laid[p.Block]
	}
	b := &e.st.Doc.Blocks[p.Block]
	if v == nil || !b.Kind.HasText() || len(v.lines) == 0 {
		if end {
			p.Offset = e.st.Doc.BlockLen(p.Block)
		} else {
			p.Offset = 0
		}
		return p
	}
	li, _ := stopFor(v.lines, p.Offset)
	lo, hi := b.Len(), 0
	for _, s := range v.lines[li].Stops {
		lo, hi = min(lo, s.Rune), max(hi, s.Rune)
	}
	if end {
		// A line that wraps ends before its last stop's space or newline
		// would move the caret to the next line.
		if li < len(v.lines)-1 && hi > lo {
			hi--
		}
		p.Offset = min(hi, b.Len())
	} else {
		p.Offset = lo
	}
	return p
}

// selectionRects returns the selection's boxes in block i, in content
// coordinates.
func (e *RichEditor) selectionRects(i int, from, to int) []image.Rectangle {
	v := e.laid[i]
	if v == nil || from >= to {
		return nil
	}
	o := e.textOrigin(i, v)
	var out []image.Rectangle
	for li, ln := range v.lines {
		minX, maxX, any := 0, 0, false
		for k := 0; k+1 < len(ln.Stops); k++ {
			a, b := ln.Stops[k], ln.Stops[k+1]
			lo, hi := min(a.Rune, b.Rune), max(a.Rune, b.Rune)
			if lo >= from && hi <= to && lo != hi {
				x0, x1 := min(a.X, b.X), max(a.X, b.X)
				if !any {
					minX, maxX, any = x0, x1, true
				} else {
					minX, maxX = min(minX, x0), max(maxX, x1)
				}
			}
		}
		// A newline or the block's end inside the selection shows as a
		// little room past the line's end.
		last := ln.Stops[len(ln.Stops)-1]
		if to > last.Rune && from <= last.Rune && (li < len(v.lines)-1 || to > last.Rune) {
			if !any {
				minX, maxX, any = last.X, last.X, true
			}
			maxX += v.laid.Dims.Size.Y / max(len(v.lines), 1) / 3
		}
		if any {
			out = append(out, image.Rect(o.X+minX, o.Y+ln.Top, o.X+maxX, o.Y+ln.Bottom))
		}
	}
	return out
}
