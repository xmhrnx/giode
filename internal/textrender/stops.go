package textrender

import (
	"image"
	"sort"

	"gioui.org/text"
	"golang.org/x/image/math/fixed"
)

// Caret stops: the positions between glyph clusters of laid out text,
// for selecting it.

// Stop is a caret position: the number of runes before it and its x.
type Stop struct {
	Rune int
	X    int
}

// StopLine is a line of caret stops with its vertical extent.
type StopLine struct {
	Top, Bottom int
	Stops       []Stop
}

// lineStops appends the stops of a line of glyphs, offset by off and
// starting at rune r, and returns the rune count after it.
func lineStops(out *StopLine, glyphs []text.Glyph, off image.Point, r int) int {
	if len(glyphs) == 0 {
		return r
	}
	x := func(v fixed.Int26_6) int { return off.X + v.Round() }
	init := len(out.Stops) == 0
	out.Stops = append(out.Stops, Stop{Rune: r, X: x(glyphs[0].X)})
	for _, g := range glyphs {
		top := off.Y + int(g.Y) - g.Ascent.Ceil()
		bottom := off.Y + int(g.Y) + g.Descent.Ceil()
		if init {
			out.Top, out.Bottom, init = top, bottom, false
		} else {
			out.Top, out.Bottom = min(out.Top, top), max(out.Bottom, bottom)
		}
		if g.Flags&text.FlagClusterBreak != 0 && g.Flags&text.FlagTruncator == 0 {
			r += int(g.Runes)
			out.Stops = append(out.Stops, Stop{Rune: r, X: x(g.X + g.Advance)})
		}
	}
	return r
}

// StopLines returns the text's caret stops by line, in the text's
// coordinates. Rune counts are from the start of the text.
func (l *Laid) StopLines() []StopLine {
	var lines []StopLine
	r := 0
	l.EachLine(func(gs []text.Glyph, dy int) {
		var ln StopLine
		r = lineStops(&ln, gs, image.Pt(0, dy), r)
		if len(ln.Stops) > 0 {
			lines = append(lines, ln)
		}
	})
	return lines
}

// StopLines returns the rich text's caret stops by line. spanStart[i]
// is the rune offset of span i in the whole text.
func (l *RichLaid) StopLines(spanStart []int) []StopLine {
	var lines []StopLine
	baseline := 0
	for i, s := range l.Segments {
		if (len(s.Glyphs) == 0 && !s.Box) || s.Span >= len(spanStart) {
			continue
		}
		b := s.Offset.Y + s.Ascent
		if i == 0 || len(lines) == 0 || b != baseline {
			lines = append(lines, StopLine{})
			baseline = b
		}
		if s.Box {
			// A box is selected whole: stops at its two ends.
			ln := &lines[len(lines)-1]
			r := spanStart[s.Span] + s.RuneStart
			top, bottom := s.Offset.Y, s.Offset.Y+s.Size.Y
			if len(ln.Stops) == 0 {
				ln.Top, ln.Bottom = top, bottom
			} else {
				ln.Top, ln.Bottom = min(ln.Top, top), max(ln.Bottom, bottom)
			}
			ln.Stops = append(ln.Stops, Stop{Rune: r, X: s.Offset.X}, Stop{Rune: spanStart[s.Span] + s.RuneEnd, X: s.Offset.X + s.Size.X})
			continue
		}
		lineStops(&lines[len(lines)-1], s.Glyphs, s.Offset, spanStart[s.Span]+s.RuneStart)
	}
	return lines
}

// HitRune returns the rune offset of the caret stop nearest p: on the
// line p is on, or the nearest line.
func HitRune(lines []StopLine, p image.Point) int {
	if len(lines) == 0 {
		return 0
	}
	if p.Y < lines[0].Top {
		return lines[0].Stops[0].Rune
	}
	li := sort.Search(len(lines), func(i int) bool { return lines[i].Bottom > p.Y })
	if li == len(lines) {
		st := lines[len(lines)-1].Stops
		return st[len(st)-1].Rune
	}
	best, dist := 0, -1
	for _, s := range lines[li].Stops {
		d := s.X - p.X
		if d < 0 {
			d = -d
		}
		if dist < 0 || d < dist {
			best, dist = s.Rune, d
		}
	}
	return best
}

// RangeRects returns the rectangles covering runes [a, b), a line each.
func RangeRects(lines []StopLine, a, b int) []image.Rectangle {
	var rs []image.Rectangle
	if a >= b {
		return nil
	}
	for _, ln := range lines {
		st := ln.Stops
		if len(st) == 0 || st[len(st)-1].Rune <= a || st[0].Rune >= b {
			// A line break glyph ends a line with a stop past its last
			// visible rune, so a range ending there still stops at it.
			continue
		}
		x0, x1 := st[0].X, st[0].X
		set := false
		for _, s := range st {
			if s.Rune < a || s.Rune > b {
				continue
			}
			if !set {
				x0, x1, set = s.X, s.X, true
			}
			x0, x1 = min(x0, s.X), max(x1, s.X)
		}
		if set && x1 > x0 {
			rs = append(rs, image.Rect(x0, ln.Top, x1, ln.Bottom))
		}
	}
	return rs
}
