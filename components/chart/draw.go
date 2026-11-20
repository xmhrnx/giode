package chart

import (
	"image"
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/internal/textrender"
	"github.com/xmhrnx/giode/styles/properties"
)

// palette is the chart's colors for a frame.
type palette struct {
	text, muted, grid, axis, surface, border, bg color.NRGBA
}

func themePalette() palette {
	c := properties.CalcColor
	p := palette{text: c("theme.text"), muted: c("theme.muted"), border: c("theme.border"),
		surface: c("theme.surface"), bg: c("theme.background")}
	p.grid = p.border
	p.grid.A = 0x70
	p.axis = p.border
	return p
}

// seriesColor is series i's color.
func (c *Chart) seriesColor(i int) color.NRGBA {
	if i < len(c.series) && c.series[i].Color != "" {
		return properties.CalcColor(c.series[i].Color)
	}
	return categoryColor(i)
}

// categoryColor is the theme's i-th chart color.
func categoryColor(i int) color.NRGBA {
	return properties.CalcColor("theme.chart" + itoa(i+1))
}

func itoa(i int) string {
	if i < 10 {
		return string(rune('0' + i))
	}
	return itoa(i/10) + string(rune('0'+i%10))
}

// label is laid out text.
type label struct {
	l *textrender.RichLaid
}

func (l label) size() image.Point { return l.l.Dims.Size }

func text(gtx layout.Context, s string, size unit.Sp, w font.Weight) label {
	g := gtx
	g.Constraints = layout.Constraints{Max: image.Pt(1<<20, 1<<20)}
	l := textrender.LayoutRich(fonts.Shaper(), g, textrender.RichParams{},
		[]textrender.Span{{Font: fonts.ForText(font.Font{Weight: w}, s), Size: size, Content: s}})
	return label{l}
}

// draw paints the label with its top left at p.
func (l label) draw(gtx layout.Context, p image.Point, col color.NRGBA) {
	st := op.Offset(p).Push(gtx.Ops)
	l.l.Draw(gtx.Ops, fonts.Shaper(), func(int) color.NRGBA { return col })
	st.Pop()
}

// drawAt paints the label aligned at p: ax and ay are 0 start, 0.5
// middle, 1 end.
func (l label) drawAt(gtx layout.Context, p image.Point, ax, ay float64, col color.NRGBA) image.Rectangle {
	s := l.size()
	tl := image.Pt(p.X-int(float64(s.X)*ax), p.Y-int(float64(s.Y)*ay))
	l.draw(gtx, tl, col)
	return image.Rectangle{Min: tl, Max: tl.Add(s)}
}

func fill(gtx layout.Context, r image.Rectangle, c color.NRGBA) {
	if c.A == 0 || r.Empty() {
		return
	}
	paint.FillShape(gtx.Ops, c, clip.Rect(r).Op())
}

func fillRound(gtx layout.Context, r image.Rectangle, rad int, c color.NRGBA) {
	if c.A == 0 || r.Empty() {
		return
	}
	paint.FillShape(gtx.Ops, c, clip.UniformRRect(r, rad).Op(gtx.Ops))
}

func alpha(c color.NRGBA, a uint8) color.NRGBA {
	c.A = uint8(int(c.A) * int(a) / 255)
	return c
}

// polyline strokes the points.
func polyline(gtx layout.Context, pts []f32.Point, width float32, c color.NRGBA, smooth, dashed bool) {
	if len(pts) < 2 {
		return
	}
	if dashed {
		dashLine(gtx, pts, width, c)
		return
	}
	var p clip.Path
	p.Begin(gtx.Ops)
	tracePath(&p, pts, smooth)
	paint.FillShape(gtx.Ops, c, clip.Stroke{Path: p.End(), Width: width}.Op())
}

// tracePath adds the points to p, as a curve through them when smooth.
func tracePath(p *clip.Path, pts []f32.Point, smooth bool) {
	p.MoveTo(pts[0])
	if !smooth || len(pts) < 3 {
		for _, q := range pts[1:] {
			p.LineTo(q)
		}
		return
	}
	// A monotone-ish Catmull-Rom curve, its handles kept within each
	// segment's height so it doesn't overshoot much.
	for i := 0; i+1 < len(pts); i++ {
		p0 := pts[max(i-1, 0)]
		p1, p2 := pts[i], pts[i+1]
		p3 := pts[min(i+2, len(pts)-1)]
		c1 := p1.Add(p2.Sub(p0).Mul(1.0 / 6))
		c2 := p2.Sub(p3.Sub(p1).Mul(1.0 / 6))
		c1.Y = clampF(c1.Y, p1.Y, p2.Y)
		c2.Y = clampF(c2.Y, p1.Y, p2.Y)
		p.CubeTo(c1, c2, p2)
	}
}

func clampF(v, a, b float32) float32 {
	lo, hi := min(a, b), max(a, b)
	return min(max(v, lo), hi)
}

func dashLine(gtx layout.Context, pts []f32.Point, width float32, c color.NRGBA) {
	dash, gap := 6*width, 4*width
	var p clip.Path
	p.Begin(gtx.Ops)
	on, left := true, dash
	for i := 0; i+1 < len(pts); i++ {
		a, b := pts[i], pts[i+1]
		d := b.Sub(a)
		n := float32(math.Hypot(float64(d.X), float64(d.Y)))
		pos := float32(0)
		for pos < n {
			step := min(left, n-pos)
			q0 := a.Add(d.Mul(pos / n))
			q1 := a.Add(d.Mul((pos + step) / n))
			if on {
				p.MoveTo(q0)
				p.LineTo(q1)
			}
			pos += step
			left -= step
			if left <= 0 {
				on = !on
				if on {
					left = dash
				} else {
					left = gap
				}
			}
		}
	}
	paint.FillShape(gtx.Ops, c, clip.Stroke{Path: p.End(), Width: width}.Op())
}

// area fills between the top points and the bottom ones (in reverse).
func area(gtx layout.Context, top, bottom []f32.Point, c color.NRGBA, smooth bool) {
	if len(top) < 2 {
		return
	}
	var p clip.Path
	p.Begin(gtx.Ops)
	tracePath(&p, top, smooth)
	rev := make([]f32.Point, len(bottom))
	for i, q := range bottom {
		rev[len(bottom)-1-i] = q
	}
	p.LineTo(rev[0])
	if smooth && len(rev) >= 3 {
		var q clip.Path
		_ = q
		for i := 0; i+1 < len(rev); i++ {
			p0 := rev[max(i-1, 0)]
			p1, p2 := rev[i], rev[i+1]
			p3 := rev[min(i+2, len(rev)-1)]
			c1 := p1.Add(p2.Sub(p0).Mul(1.0 / 6))
			c2 := p2.Sub(p3.Sub(p1).Mul(1.0 / 6))
			c1.Y = clampF(c1.Y, p1.Y, p2.Y)
			c2.Y = clampF(c2.Y, p1.Y, p2.Y)
			p.CubeTo(c1, c2, p2)
		}
	} else {
		for _, q := range rev[1:] {
			p.LineTo(q)
		}
	}
	p.Close()
	paint.FillShape(gtx.Ops, c, clip.Outline{Path: p.End()}.Op())
}

// circle fills a circle.
func circle(gtx layout.Context, center f32.Point, r float32, c color.NRGBA) {
	rect := image.Rect(int(center.X-r), int(center.Y-r), int(math.Ceil(float64(center.X+r))), int(math.Ceil(float64(center.Y+r))))
	paint.FillShape(gtx.Ops, c, clip.Ellipse(rect).Op(gtx.Ops))
}

// arcPath traces a ring segment from angle a0 to a1 (radians, clockwise
// from 3 o'clock) between radii r0 and r1.
func arcPath(gtx layout.Context, center f32.Point, r0, r1 float32, a0, a1 float64) clip.PathSpec {
	var p clip.Path
	p.Begin(gtx.Ops)
	steps := max(int(math.Abs(a1-a0)/(math.Pi/90)), 2)
	pt := func(r float32, a float64) f32.Point {
		return f32.Pt(center.X+r*float32(math.Cos(a)), center.Y+r*float32(math.Sin(a)))
	}
	p.MoveTo(pt(r1, a0))
	for i := 1; i <= steps; i++ {
		p.LineTo(pt(r1, a0+(a1-a0)*float64(i)/float64(steps)))
	}
	if r0 <= 0 {
		p.LineTo(center)
	} else {
		for i := steps; i >= 0; i-- {
			p.LineTo(pt(r0, a0+(a1-a0)*float64(i)/float64(steps)))
		}
	}
	p.Close()
	return p.End()
}

func fillArc(gtx layout.Context, center f32.Point, r0, r1 float32, a0, a1 float64, c color.NRGBA) {
	if a1-a0 <= 0 {
		return
	}
	paint.FillShape(gtx.Ops, c, clip.Outline{Path: arcPath(gtx, center, r0, r1, a0, a1)}.Op())
}

// lerpColor mixes a toward b by t.
func lerpColor(a, b color.NRGBA, t float64) color.NRGBA {
	m := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return color.NRGBA{m(a.R, b.R), m(a.G, b.G), m(a.B, b.B), m(a.A, b.A)}
}

func pt(x, y float64) f32.Point { return f32.Pt(float32(x), float32(y)) }
