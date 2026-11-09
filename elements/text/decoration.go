package text

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"

	"github.com/xmhrnx/giode/internal/textrender"
	"github.com/xmhrnx/giode/styles/properties"
)

// decorated reports whether deco draws a line.
func decorated(deco properties.TextDecoration) bool {
	return deco != "" && deco != properties.TextDecorationNone
}

// decorationRect is the line of an underline or line-through under (or
// through) the text from x0 to x1 on a baseline, with the line's ascent
// and descent: a line-through at about half the x-height, an underline
// a little below the baseline.
func decorationRect(deco properties.TextDecoration, x0, x1, baseline, ascent, descent, th int) image.Rectangle {
	var y int
	switch deco {
	case properties.TextDecorationLineThrough:
		y = baseline - (ascent*3+5)/10 - th/2
	default:
		y = baseline + max(th, (descent*35+50)/100)
	}
	return image.Rect(x0, y, x1, y+th)
}

func fillRect(gtx layout.Context, r image.Rectangle, col color.NRGBA) {
	if r.Dx() <= 0 || r.Dy() <= 0 {
		return
	}
	defer clip.Rect(r).Push(gtx.Ops).Pop()
	paint.ColorOp{Color: col}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
}

// decorationLines returns the lines of a text's decoration: one per line
// of text, from its first glyph with ink to its last, so it neither
// runs past the text in a wider box nor under trailing spaces.
func decorationLines(l *textrender.Laid, deco properties.TextDecoration, th int) []image.Rectangle {
	var out []image.Rectangle
	l.EachLine(func(gs []text.Glyph, dy int) {
		x0, x1, ascent, descent, baseline, ink := 0, 0, 0, 0, 0, false
		for _, g := range gs {
			ascent, descent = max(ascent, g.Ascent.Ceil()), max(descent, g.Descent.Ceil())
			baseline = dy + int(g.Y)
			if g.Bounds.Empty() {
				continue // a space
			}
			l, r := (g.X + g.Bounds.Min.X).Floor(), (g.X + g.Bounds.Max.X).Ceil()
			if a := (g.X + g.Advance).Round(); a > r {
				r = a
			}
			if !ink {
				x0, x1, ink = l, r, true
			} else {
				x0, x1 = min(x0, l), max(x1, r)
			}
		}
		if ink {
			out = append(out, decorationRect(deco, x0, x1, baseline, ascent, descent, th))
		}
	})
	return out
}

// paintDecoration draws an underline or line-through under each line of
// laid out text.
func paintDecoration(gtx layout.Context, l *textrender.Laid, deco properties.TextDecoration, col color.NRGBA) {
	if !decorated(deco) || l == nil {
		return
	}
	for _, r := range decorationLines(l, deco, max(1, gtx.Dp(1))) {
		fillRect(gtx, r, col)
	}
}

// paintSegmentDecoration draws a rich text segment's decoration, size
// wide, on its baseline (ascent from its top).
func paintSegmentDecoration(gtx layout.Context, size image.Point, ascent int, deco properties.TextDecoration, col color.NRGBA) {
	if !decorated(deco) {
		return
	}
	descent := max(0, size.Y-ascent)
	fillRect(gtx, decorationRect(deco, 0, size.X, ascent, ascent, descent, max(1, gtx.Dp(1))), col)
}
