package textrender

// The rich text layout follows gioui.org/x/styledtext (Unlicense OR
// MIT), collecting the shaped segments instead of painting them, so a
// layout can be cached and drawn every frame without shaping.

import (
	"hash/maphash"
	"image"
	"image/color"
	"math"
	"strings"
	"sync"
	"unicode/utf8"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"golang.org/x/image/math/fixed"

	"github.com/xmhrnx/giode/fonts"
)

// Span is a run of text with its own style, or, with Box set, an
// inline box: something drawn in the line (a formula), Box wide and
// tall, its baseline Ascent from its top. A box stands for Content,
// its text when selected, and wraps as a word does.
type Span struct {
	Font    font.Font
	Size    unit.Sp
	Content string
	Box     image.Point
	Ascent  int
}

// IsBox reports whether the span is an inline box.
func (s Span) IsBox() bool { return s.Box != (image.Point{}) }

// RichParams are the paragraph-wide settings of rich text.
type RichParams struct {
	Alignment       text.Alignment
	LineHeight      unit.Sp
	LineHeightScale float32
	// MaxLines, when above zero, stops after that many lines; with
	// Ellipsis, the last ends in "…" when text was left out.
	MaxLines int
	Ellipsis bool
}

// Segment is a piece of one span on one line.
type Segment struct {
	// Span indexes the spans the text was laid out from.
	Span int
	// Offset is the segment's top left corner in the text; Size its
	// size; Ascent its baseline from its top.
	Offset image.Point
	Size   image.Point
	Ascent int
	// Glyphs are the segment's glyphs, relative to Offset.
	Glyphs []text.Glyph
	// Runes are the span's runes the segment covers, from its start.
	RuneStart, RuneEnd int
	// Box marks an inline box's segment, which has no glyphs.
	Box bool
}

// RichLaid is laid out rich text.
type RichLaid struct {
	Segments []Segment
	Dims     layout.Dimensions
	keys     []uint64
	used     uint64
}

var richLayouts struct {
	sync.Mutex
	m map[uint64]*richEntry
}

type richEntry struct {
	spans []Span
	*RichLaid
}

// LayoutRich lays out spans as styledtext.TextStyle would, from cache
// when the spans, settings and constraints are the same as before.
func LayoutRich(sh *text.Shaper, gtx layout.Context, p RichParams, spans []Span) *RichLaid {
	key := richKey(gtx, p, spans)
	now := frame.Load()
	richLayouts.Lock()
	if e, ok := richLayouts.m[key]; ok && sameSpans(e.spans, spans) {
		e.used = now
		richLayouts.Unlock()
		return e.RichLaid
	}
	richLayouts.Unlock()
	l := layoutRich(sh, gtx, p, spans)
	l.used = now
	richLayouts.Lock()
	if richLayouts.m == nil {
		richLayouts.m = make(map[uint64]*richEntry)
	}
	richLayouts.m[key] = &richEntry{spans: append([]Span(nil), spans...), RichLaid: l}
	richLayouts.Unlock()
	return l
}

func richKey(gtx layout.Context, p RichParams, spans []Span) uint64 {
	var h maphash.Hash
	h.SetSeed(seed)
	var b [8]byte
	w := func(v uint64) { putU64(b[:], v); h.Write(b[:]) }
	w(uint64(gtx.Constraints.Max.X))
	w(uint64(gtx.Constraints.Max.Y))
	w(uint64(math.Float32bits(gtx.Metric.PxPerSp)))
	w(uint64(p.Alignment))
	w(uint64(math.Float32bits(float32(p.LineHeight))))
	w(uint64(math.Float32bits(p.LineHeightScale)))
	w(uint64(p.MaxLines))
	if p.Ellipsis {
		w(1)
	}
	h.WriteString(gtx.Locale.Language)
	for _, s := range spans {
		h.WriteString(string(s.Font.Typeface))
		w(uint64(s.Font.Weight)<<16 | uint64(s.Font.Style))
		w(uint64(math.Float32bits(float32(s.Size))))
		w(uint64(len(s.Content)))
		h.WriteString(s.Content)
		w(uint64(s.Box.X)<<32 | uint64(uint32(s.Box.Y)))
		w(uint64(s.Ascent))
	}
	return h.Sum64()
}

func sameSpans(a, b []Span) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Draw paints the segments, each in the color colorOf returns for its
// span.
func (l *RichLaid) Draw(ops *op.Ops, sh *text.Shaper, colorOf func(span int) color.NRGBA) {
	if len(l.keys) != len(l.Segments) {
		l.keys = make([]uint64, len(l.Segments))
		for i, s := range l.Segments {
			l.keys[i] = runHash(s.Glyphs)
		}
	}
	for i, s := range l.Segments {
		if s.Box {
			continue
		}
		st := op.Offset(s.Offset).Push(ops)
		drawRunKeyed(ops, sh, s.Glyphs, colorOf(s.Span), l.keys[i])
		st.Pop()
	}
}

type spanState struct {
	Span
	idx       int
	runeStart int
	// lines are the span's remaining lines, shaped at once when it
	// started a line (see spanLines); the first is this line's.
	lines []spanResult
}

type spanResult struct {
	glyphs           []text.Glyph
	width, height    int
	ascent           int
	runes            int
	multiLine        bool
	endedWithNewline bool
}

func shapeSpan(sh *text.Shaper, gtx layout.Context, p RichParams, maxWidth int, span spanState, truncate bool) (spanResult, richIterator) {
	maxLines := 0
	if truncate {
		maxLines = 1
	}
	fonts.Shaped(len(span.Content))
	sh.LayoutString(text.Parameters{
		Font:            span.Font,
		PxPerEm:         fixed.I(gtx.Sp(span.Size)),
		MaxLines:        maxLines,
		MaxWidth:        maxWidth,
		Truncator:       "​",
		Locale:          gtx.Locale,
		WrapPolicy:      text.WrapWords,
		LineHeight:      fixed.I(gtx.Sp(p.LineHeight)),
		LineHeightScale: p.LineHeightScale,
	}, span.Content)
	ti := richIterator{viewport: image.Rectangle{Max: gtx.Constraints.Max}, maxLines: 1}
	var res spanResult
	for g, ok := sh.NextGlyph(); ok; g, ok = sh.NextGlyph() {
		visibleOrBefore := ti.processGlyph(g)
		if ti.visible {
			if !ti.init {
				ti.firstX = g.X
				ti.init = true
			}
			g.X -= ti.firstX
			res.glyphs = append(res.glyphs, g)
		}
		if !visibleOrBefore {
			break
		}
	}
	return res, ti
}

func layoutSpan(sh *text.Shaper, gtx layout.Context, p RichParams, maxWidth int, span spanState) spanResult {
	res, ti := shapeSpan(sh, gtx, p, maxWidth, span, true)
	runes := ti.runes
	multiLine := runes < utf8.RuneCountInString(span.Content)
	endedWithNewline := ti.hasNewline
	if multiLine {
		i := 0
		for n := 0; n < runes; n++ {
			_, sz := utf8.DecodeRuneInString(span.Content[i:])
			i += sz
		}
		first, _ := utf8.DecodeRuneInString(span.Content[i:])
		if first == '\n' {
			endedWithNewline = true
			runes++
		} else if runes == 0 {
			// Nothing fit on the line and it wasn't a hard newline: wrap
			// without truncation to find the word that doesn't fit.
			res, ti = shapeSpan(sh, gtx, p, maxWidth, span, false)
			runes = ti.runes
			multiLine = runes < utf8.RuneCountInString(span.Content)
			endedWithNewline = ti.hasNewline
		}
	}
	res.width, res.height = ti.bounds.Dx(), ti.bounds.Dy()
	res.ascent = ti.baseline
	res.runes = runes
	res.multiLine = multiLine
	res.endedWithNewline = endedWithNewline
	return res
}

func layoutRich(sh *text.Shaper, gtx layout.Context, p RichParams, in []Span) *RichLaid {
	spans := make([]spanState, len(in))
	for i, s := range in {
		spans[i] = spanState{Span: s, idx: i}
	}
	scale := p.LineHeightScale
	lineHeightPx := gtx.Sp(p.LineHeight)
	if scale == 0 {
		scale = 1.2
	}
	var (
		lineDims image.Point
		overall  image.Point
		line     []Segment
		out      = &RichLaid{}
		// lines counts the lines placed; lastLine is where the last
		// one's segments start in out.Segments.
		lines, lastLine int
		truncated       bool
	)
	// endLine places the line's segments, the line as tall as its text,
	// or as its boxes where they reach past the text and its leading.
	endLine := func() {
		textAsc, textDesc, asc, desc, hasText := 0, 0, 0, 0, false
		for _, s := range line {
			asc, desc = max(asc, s.Ascent), max(desc, s.Size.Y-s.Ascent)
			if !s.Box {
				hasText = true
				textAsc, textDesc = max(textAsc, s.Ascent), max(textDesc, s.Size.Y-s.Ascent)
			}
		}
		h := lineDims.Y
		if p.LineHeight != 0 {
			h = lineHeightPx
		}
		adv := int(float32(h) * scale)
		extraTop, extraBottom := 0, 0
		if hasText {
			// Boxes may use half the leading on each side before the
			// line grows.
			half := max(adv-textAsc-textDesc, 0) / 2
			extraTop = max(asc-textAsc-half, 0)
			extraBottom = max(desc-textDesc-half, 0)
		} else {
			adv = asc + desc
		}
		if len(line) > 0 {
			last := line[len(line)-1]
			lineWidth := last.Offset.X + last.Size.X
			pad := 0
			if lineWidth < gtx.Constraints.Max.X {
				switch p.Alignment {
				case text.Middle:
					pad = (gtx.Constraints.Max.X - lineWidth) / 2
				case text.End:
					pad = gtx.Constraints.Max.X - lineWidth
				}
			}
			// Text keeps the line's top, as it always has; boxes sit on
			// the baseline of its tallest face.
			base := overall.Y + extraTop + textAsc
			if !hasText {
				base = overall.Y + asc
			}
			lastLine = len(out.Segments)
			for _, s := range line {
				s.Offset.X += pad
				if s.Box {
					s.Offset.Y = base - s.Ascent
				} else {
					s.Offset.Y = overall.Y + extraTop
				}
				out.Segments = append(out.Segments, s)
			}
		}
		line = line[:0]
		overall.Y += adv + extraTop + extraBottom
		lineDims = image.Point{}
		lines++
	}
	for i := 0; i < len(spans); i++ {
		if p.MaxLines > 0 && lines >= p.MaxLines {
			truncated = true
			break
		}
		span := spans[i]
		maxWidth := gtx.Constraints.Max.X - lineDims.X
		if span.IsBox() {
			force := lineDims.X > 0 && span.Box.X > maxWidth
			if !force {
				line = append(line, Segment{
					Span:      span.idx,
					Offset:    image.Point{X: lineDims.X},
					Size:      span.Box,
					Ascent:    span.Ascent,
					RuneStart: span.runeStart,
					RuneEnd:   span.runeStart + utf8.RuneCountInString(span.Content),
					Box:       true,
				})
				lineDims.X += span.Box.X
				overall.X = max(overall.X, lineDims.X)
			}
			if force || i == len(spans)-1 {
				endLine()
			}
			if force {
				i--
			}
			continue
		}
		if lineDims.X == 0 && span.lines == nil && len(span.Content) > spanLinesMin {
			// At the start of a line, the span's lines are all full
			// width: shape it once and take them in turn, rather than
			// shaping what's left of it for every line.
			span.lines = spanLines(sh, gtx, p, maxWidth, span)
		}
		var res spanResult
		if len(span.lines) > 0 {
			res = span.lines[0]
		} else {
			res = layoutSpan(sh, gtx, p, maxWidth, span)
		}
		if res.multiLine || res.endedWithNewline {
			// A line that ends here ends at its last visible glyph:
			// spaces before a wrap or a newline don't count, however
			// the line was shaped.
			res.width = min(res.width, inkWidth(res.glyphs))
		}
		forceToNextLine := lineDims.X > 0 && res.width > maxWidth
		if !forceToNextLine {
			line = append(line, Segment{
				Span:      span.idx,
				Offset:    image.Point{X: lineDims.X},
				Size:      image.Point{X: res.width, Y: res.height},
				Ascent:    res.ascent,
				Glyphs:    res.glyphs,
				RuneStart: span.runeStart,
				RuneEnd:   span.runeStart + res.runes,
			})
			lineDims.X += res.width
			lineDims.Y = max(lineDims.Y, res.height)
			overall.X = max(overall.X, lineDims.X)
		}
		if res.multiLine || res.endedWithNewline || i == len(spans)-1 || forceToNextLine {
			endLine()
		}
		if res.multiLine && !forceToNextLine {
			spans = append(spans, spanState{})
			copy(spans[i+2:], spans[i+1:])
			byteLen := 0
			for n := 0; n < res.runes; n++ {
				_, sz := utf8.DecodeRuneInString(span.Content[byteLen:])
				byteLen += sz
			}
			span.Content = span.Content[byteLen:]
			span.runeStart += res.runes
			if len(span.lines) > 0 {
				span.lines = span.lines[1:]
			}
			spans[i+1] = span
		} else if forceToNextLine {
			i--
		}
	}
	if truncated && p.Ellipsis {
		ellipsize(sh, gtx, p, in, out, lastLine)
	}
	out.Dims = layout.Dimensions{Size: gtx.Constraints.Constrain(overall)}
	return out
}

// ellipsize ends the last line, from out.Segments[from], with an
// ellipsis in its last text span's font, dropping glyphs (and boxes)
// from its end until the ellipsis fits the width.
func ellipsize(sh *text.Shaper, gtx layout.Context, p RichParams, in []Span, out *RichLaid, from int) {
	segs := out.Segments[from:]
	last := -1
	for i := len(segs) - 1; i >= 0; i-- {
		if !segs[i].Box {
			last = i
			break
		}
	}
	if last < 0 {
		return
	}
	ref := segs[last]
	ell, _ := shapeSpan(sh, gtx, p, gtx.Constraints.Max.X, spanState{Span: Span{Font: in[ref.Span].Font, Size: in[ref.Span].Size, Content: "…"}}, true)
	ellW := 0
	for _, g := range ell.glyphs {
		ellW = max(ellW, (g.X + g.Advance).Ceil())
	}
	limit := gtx.Constraints.Max.X - ellW
	// Drop what reaches past limit, glyph by glyph.
	end := 0
	for len(segs) > 0 {
		s := &segs[len(segs)-1]
		if s.Box {
			if s.Offset.X+s.Size.X <= limit {
				end = s.Offset.X + s.Size.X
				break
			}
			segs = segs[:len(segs)-1]
			continue
		}
		n := len(s.Glyphs)
		for n > 0 && s.Offset.X+(s.Glyphs[n-1].X+s.Glyphs[n-1].Advance).Ceil() > limit {
			n--
		}
		if n == 0 {
			segs = segs[:len(segs)-1]
			continue
		}
		s.Glyphs = s.Glyphs[:n:n]
		s.Size.X = (s.Glyphs[n-1].X + s.Glyphs[n-1].Advance).Ceil()
		end = s.Offset.X + s.Size.X
		break
	}
	out.Segments = append(out.Segments[:from], segs...)
	out.Segments = append(out.Segments, Segment{
		Span:      ref.Span,
		Offset:    image.Point{X: end, Y: ref.Offset.Y},
		Size:      image.Point{X: ellW, Y: ref.Size.Y},
		Ascent:    ref.Ascent,
		Glyphs:    ell.glyphs,
		RuneStart: ref.RuneEnd,
		RuneEnd:   ref.RuneEnd,
	})
}

// spanLines shapes span once, starting a line maxWidth wide, and
// returns its lines as layoutSpan would have, line by line: each line's
// glyphs, size, baseline and runes, the glyphs placed as if the line
// were shaped on its own - all but the last, which the next span may
// continue. It returns none when a line wouldn't come out the same (a
// word too long for the line), for layoutSpan to handle line by line.
func spanLines(sh *text.Shaper, gtx layout.Context, p RichParams, maxWidth int, span spanState) []spanResult {
	if len(span.Content) < streamMin || hasRTL(span.Content) {
		return shapeLines(sh, gtx, p, maxWidth, span)
	}
	// A long span is most likely text streaming in, a little longer
	// every frame: its lines but the last stay as they were, so they're
	// reused and only the text from there on is shaped.
	key := streamKey{
		font: span.Font, size: span.Size, pxPerSp: gtx.Metric.PxPerSp, width: maxWidth,
		lineHeight: p.LineHeight, scale: p.LineHeightScale, lang: gtx.Locale.Language,
		head: span.Content[:streamHead],
	}
	now := frame.Load()
	streams.Lock()
	e := streams.m[key]
	streams.Unlock()
	var lines []spanResult
	var ends []int
	if e != nil && len(e.lines) > 1 {
		// Keep all but the last line shaped before, as a margin: it
		// may end where the text was cut.
		k := len(e.lines) - 1
		if cut := e.ends[k-1]; strings.HasPrefix(span.Content, e.content[:cut]) {
			rest := span
			rest.Content = span.Content[cut:]
			more := shapeLines(sh, gtx, p, maxWidth, rest)
			if len(more) > 0 || len(rest.Content) <= streamMin {
				lines = append(append(make([]spanResult, 0, k+len(more)), e.lines[:k]...), more...)
				ends = append(make([]int, 0, len(lines)), e.ends[:k]...)
				ends = lineEnds(ends, span.Content, cut, more)
			}
		}
	}
	if lines == nil {
		lines = shapeLines(sh, gtx, p, maxWidth, span)
		ends = lineEnds(nil, span.Content, 0, lines)
	}
	if len(lines) > 0 {
		streams.Lock()
		if streams.m == nil {
			streams.m = make(map[streamKey]*streamEntry)
		}
		streams.m[key] = &streamEntry{content: span.Content, lines: lines, ends: ends, used: now}
		streams.Unlock()
	}
	return lines
}

// streamMin is the length in bytes from which a span's lines are kept
// for reuse when it grows; streamHead how much of its start tells
// spans of the same style apart.
const (
	streamMin  = 512
	streamHead = 64
)

type streamKey struct {
	font       font.Font
	size       unit.Sp
	pxPerSp    float32
	width      int
	lineHeight unit.Sp
	scale      float32
	lang       string
	head       string
}

// streamEntry is a long span's text and its lines but the last, ends
// the byte offset in the text where each line ends.
type streamEntry struct {
	content string
	lines   []spanResult
	ends    []int
	used    uint64
}

var streams struct {
	sync.Mutex
	m map[streamKey]*streamEntry
}

// lineEnds appends to ends where each of lines ends in s, the lines
// starting at byte from.
func lineEnds(ends []int, s string, from int, lines []spanResult) []int {
	i := from
	for _, ln := range lines {
		for n := 0; n < ln.runes && i < len(s); n++ {
			_, sz := utf8.DecodeRuneInString(s[i:])
			i += sz
		}
		ends = append(ends, i)
	}
	return ends
}

// hasRTL reports whether s holds right-to-left letters, whose lines a
// shape of part of the paragraph may order differently.
func hasRTL(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 0xD6 {
			continue
		}
		r, _ := utf8.DecodeRuneInString(s[i:])
		if r >= 0x0590 && r <= 0x08FF || r >= 0xFB1D && r <= 0xFEFF || r >= 0x10800 && r <= 0x10FFF || r >= 0x1E800 && r <= 0x1EFFF {
			return true
		}
	}
	return false
}

// shapeLines is spanLines without reuse.
func shapeLines(sh *text.Shaper, gtx layout.Context, p RichParams, maxWidth int, span spanState) []spanResult {
	fonts.Shaped(len(span.Content))
	sh.LayoutString(text.Parameters{
		Font:            span.Font,
		PxPerEm:         fixed.I(gtx.Sp(span.Size)),
		MaxWidth:        maxWidth,
		Truncator:       "​",
		Locale:          gtx.Locale,
		WrapPolicy:      text.WrapWords,
		LineHeight:      fixed.I(gtx.Sp(p.LineHeight)),
		LineHeightScale: p.LineHeightScale,
	}, span.Content)
	var (
		lines []spanResult
		ti    richIterator
		cur   spanResult
		// shift moves this line's glyphs up to where the first line's
		// are: a line shaped on its own starts there.
		shift    int32
		first    = true
		baseline int32
		bad      = -1
	)
	reset := func() {
		ti = richIterator{viewport: image.Rectangle{Max: gtx.Constraints.Max}, maxLines: 1}
		cur = spanResult{}
	}
	reset()
	for g, more := sh.NextGlyph(); more; g, more = sh.NextGlyph() {
		if first {
			first, baseline = false, g.Y
		}
		if ti.runes == 0 && !ti.first {
			shift = g.Y - baseline
		}
		g.Y -= shift
		ti.processGlyph(g)
		if ti.visible {
			if !ti.init {
				ti.firstX = g.X
				ti.init = true
			}
			g.X -= ti.firstX
			cur.glyphs = append(cur.glyphs, g)
		}
		if g.Flags&text.FlagLineBreak == 0 {
			continue
		}
		// The end of a line.
		cur.width, cur.height = ti.bounds.Dx(), ti.bounds.Dy()
		cur.ascent = ti.baseline
		cur.runes = ti.runes
		if g.Flags&text.FlagParagraphBreak != 0 && g.Flags&text.FlagTruncator == 0 {
			cur.endedWithNewline = true
		}
		if cur.runes == 0 && !cur.endedWithNewline {
			// Nothing fit (a word too long for the line), unless it's
			// the empty line after a final newline: the last line,
			// left to layoutSpan below.
			bad = len(lines)
		}
		cur.multiLine = true
		lines = append(lines, cur)
		reset()
	}
	if bad >= 0 && bad < len(lines)-1 || len(lines) < 2 {
		return []spanResult{}
	}
	// The last line is shaped on its own (layoutSpan), as before: the
	// next span continues it, after its trailing space, which a wrapped
	// paragraph's last line doesn't count.
	return lines[:len(lines)-1]
}

// spanLinesMin is the length in bytes above which a span starting a
// line is shaped at once (spanLines): a shorter one most likely fits on
// the line, and shaping it whole would only add a shape.
const spanLinesMin = 96

// inkWidth is the width of a line's glyphs from its left edge to the
// right edge of its rightmost glyph with a visible shape: the line
// without the blanks that end it on the right.
func inkWidth(glyphs []text.Glyph) int {
	if len(glyphs) == 0 {
		return 0
	}
	left, right := glyphs[0].X.Floor(), math.MinInt
	for _, g := range glyphs {
		left = min(left, g.X.Floor())
		if !g.Bounds.Empty() {
			right = max(right, (g.X + g.Advance).Ceil())
		}
	}
	if right == math.MinInt {
		return 0
	}
	return max(0, right-left)
}

// richIterator follows styledtext's textIterator.
type richIterator struct {
	viewport   image.Rectangle
	maxLines   int
	linesSeen  int
	init       bool
	firstX     fixed.Int26_6
	hasNewline bool
	padding    image.Rectangle
	bounds     image.Rectangle
	runes      int
	visible    bool
	first      bool
	baseline   int
}

func (it *richIterator) processGlyph(g text.Glyph) bool {
	logical := image.Rectangle{
		Min: image.Pt(g.X.Floor(), int(g.Y)-g.Ascent.Ceil()),
		Max: image.Pt((g.X + g.Advance).Ceil(), int(g.Y)+g.Descent.Ceil()),
	}
	if g.Flags&text.FlagTruncator != 0 {
		if it.runes == 0 {
			it.hasNewline = true
		}
		if !it.first {
			// A line of nothing (a newline): it's as tall as the
			// truncator. A line of glyphs is as tall as they are, as
			// when the span is shaped whole (spanLines).
			it.bounds.Min.Y = min(it.bounds.Min.Y, logical.Min.Y)
			it.bounds.Max.Y = max(it.bounds.Max.Y, logical.Max.Y)
		}
		it.visible = false
		return false
	}
	it.runes += int(g.Runes)
	it.hasNewline = it.hasNewline || (g.Flags&text.FlagLineBreak > 0 && g.Flags&text.FlagParagraphBreak > 0)
	if it.maxLines > 0 {
		if g.Flags&text.FlagLineBreak != 0 {
			it.linesSeen++
		}
		if it.linesSeen == it.maxLines && g.Flags&text.FlagParagraphBreak != 0 {
			it.visible = false
			return false
		}
	}
	if d := g.Bounds.Min.X.Floor(); d < it.padding.Min.X {
		it.padding.Min.X = d
	}
	if d := (g.Bounds.Max.X - g.Advance).Ceil(); d > it.padding.Max.X {
		it.padding.Max.X = d
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
	return !below
}
