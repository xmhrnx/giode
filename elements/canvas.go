package elements

import (
	"image"
	"math"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Canvas is an element drawn by draw, for charts, gauges, progress
// rings and other custom graphics. It's sized like a Box by st (Width,
// Height, or -1 to fill; without a size it fills what its parent
// gives it) and draw runs every frame with a *Painter for that area:
//
//	giode.Canvas(giode.Styles{Width: 96, Height: 96}, func(p *giode.Painter) {
//		c := p.Width() / 2
//		p.Arc(c, c, c-6, 8, 0, 360, "theme.border")
//		p.Arc(c, c, c-6, 8, 0, 360*progress, "theme.primary")
//	})
//
// Coordinates and lengths are in dp from the canvas's top-left corner;
// colors are any Color, theme tokens included, and "" or
// "currentColor" is the inherited text color.
func Canvas(st styles.Styles, draw func(p *Painter)) Element {
	return Box(st, Raw(func(gtx layout.Context) layout.Dimensions {
		size := gtx.Constraints.Max
		if size.X >= unboundedPx {
			size.X = gtx.Constraints.Min.X
		}
		if size.Y >= unboundedPx {
			size.Y = gtx.Constraints.Min.Y
		}
		defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
		draw(&Painter{gtx: gtx, size: size, scale: gtx.Metric.PxPerDp})
		return layout.Dimensions{Size: size}
	}))
}

// unboundedPx is where a constraint stops meaning a real size.
const unboundedPx = 1e5

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
	paint.Fill(p.gtx.Ops, ResolveColor(fill))
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

// Path returns an empty path to build and then Fill or Stroke.
func (p *Painter) Path() *Path { return &Path{p: p} }

// Path is a shape built from lines and curves, in dp.
type Path struct {
	p    *Painter
	segs []pathSeg
	open bool
}

type pathSeg struct {
	op  byte // 'M', 'L', 'Q', 'C', 'Z'
	pts [3]f32.Point
}

// MoveTo starts a new subpath at (x, y).
func (pa *Path) MoveTo(x, y float32) {
	pa.segs = append(pa.segs, pathSeg{op: 'M', pts: [3]f32.Point{pa.p.pt(x, y)}})
	pa.open = true
}

// LineTo adds a line to (x, y).
func (pa *Path) LineTo(x, y float32) {
	pa.add('L', pa.p.pt(x, y))
}

// QuadTo adds a quadratic Bézier curve through control point (cx, cy)
// to (x, y).
func (pa *Path) QuadTo(cx, cy, x, y float32) {
	pa.add('Q', pa.p.pt(cx, cy), pa.p.pt(x, y))
}

// CubeTo adds a cubic Bézier curve through two control points to
// (x, y).
func (pa *Path) CubeTo(c1x, c1y, c2x, c2y, x, y float32) {
	pa.add('C', pa.p.pt(c1x, c1y), pa.p.pt(c2x, c2y), pa.p.pt(x, y))
}

// ArcAround adds a circular arc of radius r around (cx, cy), from
// angle start through sweep (degrees clockwise from 12 o'clock),
// starting a subpath at the arc's start unless the path is already
// open, in which case a line joins it.
func (pa *Path) ArcAround(cx, cy, r, start, sweep float32) {
	pa.arc(cx, cy, r, start, sweep, pa.open)
}

func (pa *Path) arc(cx, cy, r, start, sweep float32, join bool) {
	x, y := polar(cx, cy, r, start)
	if join {
		pa.LineTo(x, y)
	} else {
		pa.MoveTo(x, y)
	}
	// Cubic Béziers of at most 45° each.
	n := int(math.Ceil(math.Abs(float64(sweep)) / 45))
	step := sweep / float32(n)
	k := 4.0 / 3 * float32(math.Tan(float64(step)*math.Pi/180/4)) * r
	a := start
	for range n {
		b := a + step
		x0, y0 := polar(cx, cy, r, a)
		x1, y1 := polar(cx, cy, r, b)
		// Tangents point clockwise: (cos a, sin a) in y-down screen
		// space for angles measured from 12 o'clock.
		ra, rb := float64(a)*math.Pi/180, float64(b)*math.Pi/180
		pa.CubeTo(
			x0+k*float32(math.Cos(ra)), y0+k*float32(math.Sin(ra)),
			x1-k*float32(math.Cos(rb)), y1-k*float32(math.Sin(rb)),
			x1, y1)
		a = b
	}
}

// Close closes the current subpath with a line to its start.
func (pa *Path) Close() {
	pa.segs = append(pa.segs, pathSeg{op: 'Z'})
	pa.open = false
}

func (pa *Path) add(op byte, pts ...f32.Point) {
	if !pa.open {
		// A path must start somewhere: at the first point.
		pa.segs = append(pa.segs, pathSeg{op: 'M', pts: [3]f32.Point{pts[len(pts)-1]}})
		pa.open = true
		if op == 'L' {
			return
		}
	}
	s := pathSeg{op: op}
	copy(s.pts[:], pts)
	pa.segs = append(pa.segs, s)
}

func (pa *Path) spec() clip.PathSpec {
	var path clip.Path
	path.Begin(pa.p.gtx.Ops)
	for _, s := range pa.segs {
		switch s.op {
		case 'M':
			path.MoveTo(s.pts[0])
		case 'L':
			path.LineTo(s.pts[0])
		case 'Q':
			path.QuadTo(s.pts[0], s.pts[1])
		case 'C':
			path.CubeTo(s.pts[0], s.pts[1], s.pts[2])
		case 'Z':
			path.Close()
		}
	}
	return path.End()
}

// Fill fills the area the path encloses (nonzero winding) with c.
func (pa *Path) Fill(c properties.Color) {
	if len(pa.segs) == 0 {
		return
	}
	defer clip.Outline{Path: pa.spec()}.Op().Push(pa.p.gtx.Ops).Pop()
	paint.Fill(pa.p.gtx.Ops, ResolveColor(c))
}

// Stroke draws the path's outline width dp wide in c.
func (pa *Path) Stroke(width float32, c properties.Color) {
	if len(pa.segs) == 0 || width <= 0 {
		return
	}
	defer clip.Stroke{Path: pa.spec(), Width: width * pa.p.scale}.Op().Push(pa.p.gtx.Ops).Pop()
	paint.Fill(pa.p.gtx.Ops, ResolveColor(c))
}
