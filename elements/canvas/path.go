package canvas

import (
	"math"

	"gioui.org/f32"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/xmhrnx/giode/elements/internal/base"
	"github.com/xmhrnx/giode/styles/properties"
)

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

// Path returns an empty path to build and then Fill or Stroke.
func (p *Painter) Path() *Path { return &Path{p: p} }

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
	paint.Fill(pa.p.gtx.Ops, base.ResolveColor(c))
}

// Stroke draws the path's outline width dp wide in c.
func (pa *Path) Stroke(width float32, c properties.Color) {
	if len(pa.segs) == 0 || width <= 0 {
		return
	}
	defer clip.Stroke{Path: pa.spec(), Width: width * pa.p.scale}.Op().Push(pa.p.gtx.Ops).Pop()
	paint.Fill(pa.p.gtx.Ops, base.ResolveColor(c))
}
