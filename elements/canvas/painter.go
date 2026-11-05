package canvas

import (
	"image"
	"math"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/xmhrnx/giode/elements/internal/base"
	"github.com/xmhrnx/giode/styles/properties"
)

// Painter draws on a Canvas, in dp.
type Painter struct {
	gtx   layout.Context
	size  image.Point
	scale float32
}

// Width returns the canvas's width in dp.
func (p *Painter) Width() float32 { return float32(p.size.X) / p.scale }

// Height returns the canvas's height in dp.
func (p *Painter) Height() float32 { return float32(p.size.Y) / p.scale }

// Context returns the Gio context, for raw drawing in pixels.
func (p *Painter) Context() layout.Context { return p.gtx }

func (p *Painter) pt(x, y float32) f32.Point { return f32.Pt(x*p.scale, y*p.scale) }

// Rect fills a rectangle with corner radius r.
func (p *Painter) Rect(x, y, w, h, r float32, fill properties.Color) {
	rect := image.Rectangle{Min: p.pt(x, y).Round(), Max: p.pt(x+w, y+h).Round()}
	defer clip.UniformRRect(rect, int(r*p.scale+0.5)).Push(p.gtx.Ops).Pop()
	paint.Fill(p.gtx.Ops, base.ResolveColor(fill))
}

// Circle fills a circle of radius r centered at (cx, cy).
func (p *Painter) Circle(cx, cy, r float32, fill properties.Color) {
	path := p.Path()
	path.ArcAround(cx, cy, r, 0, 360)
	path.Close()
	path.Fill(fill)
}

// Line strokes a straight line width wide.
func (p *Painter) Line(x1, y1, x2, y2, width float32, c properties.Color) {
	path := p.Path()
	path.MoveTo(x1, y1)
	path.LineTo(x2, y2)
	path.Stroke(width, c)
}

// Arc strokes a circular arc of radius r centered at (cx, cy), width
// wide, with round ends. Angles are in degrees clockwise from 12
// o'clock: start 0 and sweep 270 draw three quarters of a ring. A
// sweep of 360 or more draws the whole ring.
func (p *Painter) Arc(cx, cy, r, width, start, sweep float32, c properties.Color) {
	if sweep == 0 {
		return
	}
	full := sweep >= 360 || sweep <= -360
	if full {
		sweep = 360
	}
	path := p.Path()
	path.ArcAround(cx, cy, r, start, sweep)
	if full {
		path.Close()
	}
	path.Stroke(width, c)
	if !full {
		for _, a := range []float32{start, start + sweep} {
			x, y := polar(cx, cy, r, a)
			p.Circle(x, y, width/2, c)
		}
	}
}

// Sector fills the part of a ring between radii inner and outer and
// angles start and start+sweep (degrees clockwise from 12 o'clock): a
// pie slice with inner 0, a donut segment otherwise.
func (p *Painter) Sector(cx, cy, inner, outer, start, sweep float32, fill properties.Color) {
	if sweep == 0 || outer <= inner {
		return
	}
	sweep = max(-360, min(360, sweep))
	path := p.Path()
	path.ArcAround(cx, cy, outer, start, sweep)
	if inner > 0 {
		x, y := polar(cx, cy, inner, start+sweep)
		path.LineTo(x, y)
		path.arc(cx, cy, inner, start+sweep, -sweep, true)
	} else {
		path.LineTo(cx, cy)
	}
	path.Close()
	path.Fill(fill)
}

// polar returns the point at radius r and angle a (degrees clockwise
// from 12 o'clock) around (cx, cy).
func polar(cx, cy, r, a float32) (float32, float32) {
	rad := float64(a) * math.Pi / 180
	return cx + r*float32(math.Sin(rad)), cy - r*float32(math.Cos(rad))
}
