package canvas

import (
	"image"
	"math"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/elements/internal/base"
	"github.com/xmhrnx/giode/elements/text"
	"github.com/xmhrnx/giode/styles"
)

// TextAnchor is the point of a text that Painter.Text puts at (x, y).
type TextAnchor uint8

const (
	TextAnchorTopLeft TextAnchor = iota
	TextAnchorTop
	TextAnchorTopRight
	TextAnchorLeft
	TextAnchorCenter
	TextAnchorRight
	TextAnchorBottomLeft
	TextAnchorBottom
	TextAnchorBottomRight
)

// Text draws s with its anchor point at (x, y): TextAnchorCenter
// centers it on the point, TextAnchorBottom puts it above,
// TextAnchorLeft starts it there.
// st styles it like a Text element - Color (default the inherited text
// color), FontSize, FontWeight, FontFamily, Background, Padding - and
// MaxWidth (dp) wraps it. It returns the text's size in dp.
//
//	p.Circle(x, y, 4, "theme.primary")
//	p.Text(x+8, y, "M", giode.TextAnchorLeft, giode.Styles{FontSize: 13})
func (p *Painter) Text(x, y float32, s string, anchor TextAnchor, st ...styles.Styles) (w, h float32) {
	return p.TextRotated(x, y, s, anchor, 0, st...)
}

// TextRotated is Text turned by deg degrees clockwise around its anchor
// point: -90 for a y axis label read bottom to top.
func (p *Painter) TextRotated(x, y float32, s string, anchor TextAnchor, deg float32, st ...styles.Styles) (w, h float32) {
	var s0 styles.Styles
	if len(st) > 0 {
		s0 = st[0]
	}
	el := text.Text(s, s0)
	gtx := p.gtx
	maxW := 1 << 20
	if s0.MaxWidth > 0 {
		maxW = int(float32(s0.MaxWidth) * p.scale)
	}
	gtx.Constraints = layout.Constraints{Max: image.Pt(maxW, 1<<20)}
	size := base.Measure(gtx, el).Size
	dx, dy := anchorOffset(anchor, size)
	at := p.pt(x, y)
	if deg == 0 {
		// Whole pixels, for crisp glyphs.
		off := image.Pt(int(math.Round(float64(at.X)))-dx, int(math.Round(float64(at.Y)))-dy)
		defer op.Offset(off).Push(gtx.Ops).Pop()
		defer base.PushOffset(off)()
	} else {
		tr := f32.Affine2D{}.Offset(f32.Pt(float32(-dx), float32(-dy))).
			Rotate(f32.Point{}, deg*math.Pi/180).Offset(at)
		defer op.Affine(tr).Push(gtx.Ops).Pop()
	}
	el.Layout(gtx)
	return float32(size.X) / p.scale, float32(size.Y) / p.scale
}

// MeasureText returns the size Text would draw s at, in dp.
func (p *Painter) MeasureText(s string, st ...styles.Styles) (w, h float32) {
	var s0 styles.Styles
	if len(st) > 0 {
		s0 = st[0]
	}
	gtx := p.gtx
	maxW := 1 << 20
	if s0.MaxWidth > 0 {
		maxW = int(float32(s0.MaxWidth) * p.scale)
	}
	gtx.Constraints = layout.Constraints{Max: image.Pt(maxW, 1<<20)}
	size := base.Measure(gtx, text.Text(s, s0)).Size
	return float32(size.X) / p.scale, float32(size.Y) / p.scale
}

// anchorOffset is where the anchor point is in a text of size.
func anchorOffset(a TextAnchor, size image.Point) (int, int) {
	var dx, dy int
	switch a {
	case TextAnchorTop, TextAnchorCenter, TextAnchorBottom:
		dx = size.X / 2
	case TextAnchorTopRight, TextAnchorRight, TextAnchorBottomRight:
		dx = size.X
	}
	switch a {
	case TextAnchorLeft, TextAnchorCenter, TextAnchorRight:
		dy = size.Y / 2
	case TextAnchorBottomLeft, TextAnchorBottom, TextAnchorBottomRight:
		dy = size.Y
	}
	return dx, dy
}
