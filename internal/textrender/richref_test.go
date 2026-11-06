package textrender

// layoutRichRef is the rich text layout as it was before shaping each
// span once: every line shaped the rest of its span again. Tests check
// the faster layout against it.

import (
	"image"
	"unicode/utf8"

	"gioui.org/layout"
	"gioui.org/text"
	"golang.org/x/image/math/fixed"
)

type refSpanState struct {
	Span
	idx       int
	runeStart int
}

type refSpanResult struct {
	glyphs           []text.Glyph
	width, height    int
	ascent           int
	runes            int
	multiLine        bool
	endedWithNewline bool
}

func refShapeSpan(sh *text.Shaper, gtx layout.Context, p RichParams, maxWidth int, span refSpanState, truncate bool) (refSpanResult, richIterator) {
	maxLines := 0
	if truncate {
		maxLines = 1
	}
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
	var res refSpanResult
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

func refLayoutSpan(sh *text.Shaper, gtx layout.Context, p RichParams, maxWidth int, span refSpanState) refSpanResult {
	res, ti := refShapeSpan(sh, gtx, p, maxWidth, span, true)
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
			res, ti = refShapeSpan(sh, gtx, p, maxWidth, span, false)
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

func layoutRichRef(sh *text.Shaper, gtx layout.Context, p RichParams, in []Span) *RichLaid {
	spans := make([]refSpanState, len(in))
	for i, s := range in {
		spans[i] = refSpanState{Span: s, idx: i}
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
	)
	for i := 0; i < len(spans); i++ {
		span := spans[i]
		maxWidth := gtx.Constraints.Max.X - lineDims.X
		res := refLayoutSpan(sh, gtx, p, maxWidth, span)
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
				for _, s := range line {
					s.Offset.X += pad
					s.Offset.Y = overall.Y
					out.Segments = append(out.Segments, s)
				}
			}
			line = line[:0]
			h := lineDims.Y
			if p.LineHeight != 0 {
				h = lineHeightPx
			}
			overall.Y += int(float32(h) * scale)
			lineDims = image.Point{}
		}
		if res.multiLine && !forceToNextLine {
			spans = append(spans, refSpanState{})
			copy(spans[i+2:], spans[i+1:])
			byteLen := 0
			for n := 0; n < res.runes; n++ {
				_, sz := utf8.DecodeRuneInString(span.Content[byteLen:])
				byteLen += sz
			}
			span.Content = span.Content[byteLen:]
			span.runeStart += res.runes
			spans[i+1] = span
		} else if forceToNextLine {
			i--
		}
	}
	out.Dims = layout.Dimensions{Size: gtx.Constraints.Constrain(overall)}
	return out
}
