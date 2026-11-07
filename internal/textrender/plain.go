package textrender

import (
	"image"
	"image/color"
	"strings"
	"sync"
	"unicode"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/text"
	"golang.org/x/image/math/fixed"

	"github.com/xmhrnx/giode/fonts"
)

// Laid is a laid out text: its glyphs in lines, positioned as Gio's
// widget.Label positions them, and its dimensions.
type Laid struct {
	// Glyphs are the visible glyphs in order; Lines index into them.
	Glyphs []text.Glyph
	// Lines are the glyph ranges of each line, Glyphs[Lines[i][0]:Lines[i][1]].
	Lines [][2]int
	// Dims are the text's dimensions, constrained, with its baseline.
	Dims layout.Dimensions
	// Viewport is the area drawing is clipped to: the constraints,
	// grown by how far glyphs overhang.
	Viewport image.Rectangle
	// Truncated is the number of runes cut off by MaxLines.
	Truncated int

	// paras are the paragraphs of text laid out a paragraph at a time,
	// each moved down by dy and limited to its first n glyphs; Glyphs
	// and Lines are then empty (see EachLine).
	paras []paraRef
	// lineKeys cache the image key of each line.
	lineKeys []uint64
	// boundsN caches the bounds of the first boundsN.n glyphs.
	boundsN struct {
		n int
		r image.Rectangle
	}
	used uint64
}

type paraRef struct {
	l     *Laid
	dy, n int
}

// EachLine calls fn with the glyphs of each line, in order. dy moves
// the line's glyphs down from their Y.
func (l *Laid) EachLine(fn func(glyphs []text.Glyph, dy int)) {
	if l.paras == nil {
		for _, ln := range l.Lines {
			fn(l.Glyphs[ln[0]:ln[1]], 0)
		}
		return
	}
	for _, p := range l.paras {
		for _, ln := range p.l.Lines {
			s, e := ln[0], min(ln[1], p.n)
			if s < e {
				fn(p.l.Glyphs[s:e], p.dy)
			}
		}
	}
}

type layoutKey struct {
	str string
	p   text.Parameters
	max image.Point
	min image.Point
}

var layouts struct {
	sync.Mutex
	m map[layoutKey]*Laid
}

// splitMin is the length from which multi-paragraph text is laid out a
// paragraph at a time.
const splitMin = 256

// Layout lays out str as widget.Label would with the same parameters
// and constraints, from cache when it's been laid out before.
func Layout(sh *text.Shaper, p text.Parameters, cs layout.Constraints, str string) *Laid {
	p.MaxWidth, p.MinWidth = cs.Max.X, cs.Min.X
	if splittable(p, str) {
		// Built from the cached paragraphs every time: cheap, and
		// text that changes (a stream) doesn't leave a big entry per
		// version behind.
		return layoutParagraphs(sh, p, cs, str)
	}
	key := layoutKey{str: str, p: p, max: cs.Max, min: cs.Min}
	now := frame.Load()
	layouts.Lock()
	if l, ok := layouts.m[key]; ok {
		l.used = now
		layouts.Unlock()
		return l
	}
	layouts.Unlock()

	l := layoutWhole(sh, p, cs, str)
	l.used = now
	layouts.Lock()
	if layouts.m == nil {
		layouts.m = make(map[layoutKey]*Laid)
	}
	layouts.m[key] = l
	layouts.Unlock()
	return l
}

// sweepUnused drops everything the frame f didn't use: what an idle
// window keeps is what it shows.
func sweepUnused(f uint64) {
	runCache.Lock()
	for k, v := range runCache.m {
		if v.used < f {
			runCache.bytes -= v.bytes
			delete(runCache.m, k)
		}
	}
	runCache.Unlock()
	layouts.Lock()
	for k, l := range layouts.m {
		if l.used < f {
			delete(layouts.m, k)
		}
	}
	layouts.Unlock()
	richLayouts.Lock()
	for k, l := range richLayouts.m {
		if l.used < f {
			delete(richLayouts.m, k)
		}
	}
	richLayouts.Unlock()
	streams.Lock()
	for k, e := range streams.m {
		if e.used < f {
			delete(streams.m, k)
		}
	}
	streams.Unlock()
}

// sweepLayouts drops layouts unused for two seconds or so, and line
// images unused for about five.
func sweepLayouts(now uint64) {
	sweepRuns(now)
	layouts.Lock()
	for k, l := range layouts.m {
		if now-l.used > 120 {
			delete(layouts.m, k)
		}
	}
	layouts.Unlock()
	richLayouts.Lock()
	for k, l := range richLayouts.m {
		if now-l.used > 120 {
			delete(richLayouts.m, k)
		}
	}
	richLayouts.Unlock()
	streams.Lock()
	for k, e := range streams.m {
		if now-e.used > 120 {
			delete(streams.m, k)
		}
	}
	streams.Unlock()
}

// splittable reports whether str can be laid out a paragraph at a time
// with the same result: Gio aligns every paragraph to the widest, which
// only doesn't matter for start-aligned, left-to-right text.
func splittable(p text.Parameters, str string) bool {
	// A newline before the last byte: a single paragraph ends with one.
	if len(str) < splitMin || p.MaxLines > 0 || p.Alignment != text.Start || strings.IndexByte(str[:len(str)-1], '\n') < 0 {
		return false
	}
	for _, r := range str {
		if r >= 0x0590 && (unicode.Is(unicode.Hebrew, r) || unicode.Is(unicode.Arabic, r) ||
			unicode.Is(unicode.Syriac, r) || unicode.Is(unicode.Thaana, r) || unicode.Is(unicode.Nko, r)) {
			return false
		}
	}
	return true
}

// layoutParagraphs lays out each paragraph from cache and stacks them
// as Gio would: each first baseline one line height below the previous
// paragraph's last.
func layoutParagraphs(sh *text.Shaper, p text.Parameters, cs layout.Constraints, str string) *Laid {
	// Paragraphs are laid out without a height limit; the limit applies
	// to the whole.
	pcs := cs
	pcs.Min.Y, pcs.Max.Y = 0, 1<<24
	lineHeight := lineAdvance(p)
	out := &Laid{}
	var bounds, pad image.Rectangle
	first := true
	lastBaseline := 0
	baseline := 0
	for len(str) > 0 && (first || lastBaseline <= cs.Max.Y) {
		end := strings.IndexByte(str, '\n') + 1
		if end == 0 {
			end = len(str)
		}
		para := str[:end]
		str = str[end:]
		last := len(str) == 0
		pl := Layout(sh, p, pcs, para)
		n := len(pl.Glyphs)
		if n == 0 {
			continue
		}
		dy := 0
		if !first {
			dy = lastBaseline + lineHeight - int(pl.Glyphs[0].Y)
		}
		if !last && pl.Glyphs[n-1].Flags&text.FlagParagraphStart != 0 && pl.Glyphs[n-1].Runes == 0 {
			// The cursor position after a final newline: in the whole
			// text, the next paragraph's first glyph.
			n--
		}
		// As widget.Label, lines starting below the constraints are left
		// out.
		for n > 0 && int(pl.Glyphs[n-1].Y)+dy-pl.Glyphs[n-1].Ascent.Ceil() > cs.Max.Y {
			n--
		}
		if n == 0 {
			break
		}
		out.paras = append(out.paras, paraRef{l: pl, dy: dy, n: n})
		pb := pl.bounds(n).Add(image.Pt(0, dy))
		if first {
			bounds = pb
			baseline = int(pl.Glyphs[0].Y)
		} else {
			bounds = grow(bounds, pb)
		}
		lastBaseline = int(pl.Glyphs[n-1].Y) + dy
		// The overhang of glyphs past the constraints, as in
		// widget.Label.
		pad.Min = image.Pt(min(pad.Min.X, pl.Viewport.Min.X), min(pad.Min.Y, pl.Viewport.Min.Y))
		pad.Max = image.Pt(max(pad.Max.X, pl.Viewport.Max.X-pcs.Max.X), max(pad.Max.Y, pl.Viewport.Max.Y-pcs.Max.Y))
		first = false
	}
	size := cs.Constrain(bounds.Size())
	out.Dims = layout.Dimensions{Size: size, Baseline: size.Y - baseline}
	out.Viewport = image.Rectangle{Min: pad.Min, Max: cs.Max.Add(pad.Max)}
	return out
}

// bounds returns the logical bounds of the first n glyphs, as
// widget.Label computes them.
func (l *Laid) bounds(n int) image.Rectangle {
	if l.boundsN.n == n && n > 0 {
		return l.boundsN.r
	}
	defer func() { l.boundsN.n = n }()
	var b image.Rectangle
	for i, g := range l.Glyphs[:n] {
		lb := image.Rectangle{
			Min: image.Pt(g.X.Floor(), int(g.Y)-g.Ascent.Ceil()),
			Max: image.Pt((g.X + g.Advance).Ceil(), int(g.Y)+g.Descent.Ceil()),
		}
		if i == 0 {
			b = lb
		} else {
			b = grow(b, lb)
		}
	}
	l.boundsN.r = b
	return b
}

// lineAdvance is the distance between baselines Gio uses for p.
func lineAdvance(p text.Parameters) int {
	lh := p.LineHeight
	if lh == 0 {
		lh = p.PxPerEm
	}
	scale := p.LineHeightScale
	if scale == 0 {
		scale = 1.2
	}
	return fixed.Int26_6(float32(lh) / 64 * scale * 64).Round()
}

// layoutWhole lays out str in one piece, following widget.Label's
// iteration.
func layoutWhole(sh *text.Shaper, p text.Parameters, cs layout.Constraints, str string) *Laid {
	sh.LayoutString(p, str)
	fonts.Shaped(len(str))
	it := iterator{viewport: image.Rectangle{Max: cs.Max}, maxLines: p.MaxLines}
	l := &Laid{}
	lineStart := 0
	for g, ok := sh.NextGlyph(); ok; g, ok = sh.NextGlyph() {
		visibleOrBefore := it.processGlyph(g, true)
		if it.visible {
			l.Glyphs = append(l.Glyphs, g)
		}
		if g.Flags&text.FlagLineBreak != 0 || !visibleOrBefore {
			if len(l.Glyphs) > lineStart {
				l.Lines = append(l.Lines, [2]int{lineStart, len(l.Glyphs)})
			}
			lineStart = len(l.Glyphs)
		}
		if !visibleOrBefore {
			break
		}
	}
	if len(l.Glyphs) > lineStart {
		l.Lines = append(l.Lines, [2]int{lineStart, len(l.Glyphs)})
	}
	l.Viewport = image.Rectangle{Max: cs.Max}
	l.Viewport.Min = l.Viewport.Min.Add(it.padding.Min)
	l.Viewport.Max = l.Viewport.Max.Add(it.padding.Max)
	size := cs.Constrain(it.bounds.Size())
	l.Dims = layout.Dimensions{Size: size, Baseline: size.Y - it.baseline}
	l.Truncated = it.truncated
	return l
}

// Draw paints the text in col, clipped as widget.Label clips it.
func (l *Laid) Draw(ops *op.Ops, sh *text.Shaper, col color.NRGBA) {
	defer clip.Rect(l.Viewport).Push(ops).Pop()
	if l.paras == nil {
		l.drawLines(ops, sh, col, l.Lines)
		return
	}
	for _, p := range l.paras {
		st := op.Offset(image.Pt(0, p.dy)).Push(ops)
		lines := p.l.Lines
		for len(lines) > 0 && lines[len(lines)-1][0] >= p.n {
			lines = lines[:len(lines)-1]
		}
		p.l.drawLines(ops, sh, col, lines)
		st.Pop()
	}
}

// drawLines draws the given lines of l, keyed by their cached image
// keys.
func (l *Laid) drawLines(ops *op.Ops, sh *text.Shaper, col color.NRGBA, lines [][2]int) {
	if len(l.lineKeys) != len(l.Lines) {
		l.lineKeys = make([]uint64, len(l.Lines))
		for i, ln := range l.Lines {
			l.lineKeys[i] = runHash(l.Glyphs[ln[0]:ln[1]])
		}
	}
	for i, ln := range lines {
		drawRunKeyed(ops, sh, l.Glyphs[ln[0]:ln[1]], col, l.lineKeys[i])
	}
}

// Runes returns the number of runes each glyph cluster ends, for
// mapping glyphs to text: Gio puts a cluster's rune count on its last
// glyph.
func (l *Laid) Runes() int {
	n := 0
	l.EachLine(func(gs []text.Glyph, _ int) {
		for _, g := range gs {
			n += int(g.Runes)
		}
	})
	return n
}

// iterator follows widget.Label's textIterator.
type iterator struct {
	viewport  image.Rectangle
	maxLines  int
	truncated int
	linesSeen int
	padding   image.Rectangle
	bounds    image.Rectangle
	visible   bool
	first     bool
	baseline  int
}

func (it *iterator) processGlyph(g text.Glyph, ok bool) bool {
	if it.maxLines > 0 {
		if g.Flags&text.FlagTruncator != 0 && g.Flags&text.FlagClusterBreak != 0 {
			it.truncated = int(g.Runes)
		}
		if g.Flags&text.FlagLineBreak != 0 {
			it.linesSeen++
		}
		if it.linesSeen == it.maxLines && g.Flags&text.FlagParagraphBreak != 0 {
			return false
		}
	}
	if d := g.Bounds.Min.X.Floor(); d < it.padding.Min.X {
		it.padding.Min.X = d
	}
	if d := (g.Bounds.Max.X - g.Advance).Ceil(); d > it.padding.Max.X {
		it.padding.Max.X = d
	}
	if d := (g.Bounds.Min.Y + g.Ascent).Floor(); d < it.padding.Min.Y {
		it.padding.Min.Y = d
	}
	if d := (g.Bounds.Max.Y - g.Descent).Ceil(); d > it.padding.Max.Y {
		it.padding.Max.Y = d
	}
	logical := image.Rectangle{
		Min: image.Pt(g.X.Floor(), int(g.Y)-g.Ascent.Ceil()),
		Max: image.Pt((g.X + g.Advance).Ceil(), int(g.Y)+g.Descent.Ceil()),
	}
	if !it.first {
		it.first = true
		it.baseline = int(g.Y)
		it.bounds = logical
	}
	above := logical.Max.Y < it.viewport.Min.Y
	below := logical.Min.Y > it.viewport.Max.Y
	left := logical.Max.X < it.viewport.Min.X
	right := logical.Min.X > it.viewport.Max.X
	it.visible = !above && !below && !left && !right
	if it.visible {
		it.bounds = grow(it.bounds, logical)
	}
	return ok && !below
}

// grow extends r to cover s. Unlike Union it counts empty rectangles:
// a zero-width glyph still extends the text's height.
func grow(r, s image.Rectangle) image.Rectangle {
	return image.Rectangle{
		Min: image.Pt(min(r.Min.X, s.Min.X), min(r.Min.Y, s.Min.Y)),
		Max: image.Pt(max(r.Max.X, s.Max.X), max(r.Max.Y, s.Max.Y)),
	}
}
