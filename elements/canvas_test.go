package elements

import (
	"image"
	"math"
	"testing"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/styles"
)

func TestCanvasSize(t *testing.T) {
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 2}, Constraints: layout.Constraints{Max: image.Pt(500, 500)}}
	var w, h float32
	d := Canvas(styles.Styles{Width: 40, Height: 30}, func(p *Painter) {
		w, h = p.Width(), p.Height()
		p.Rect(0, 0, w, h, 4, "theme.surface")
		p.Arc(20, 15, 10, 3, 0, 270, "theme.primary")
		p.Sector(20, 15, 5, 12, 45, 90, "#ff0000")
		p.Sector(20, 15, 0, 12, 0, 360, "#ff0000")
		p.Line(0, 0, w, h, 1, "")
	}).Layout(gtx)
	if d.Size != image.Pt(80, 60) || w != 40 || h != 30 {
		t.Errorf("size %v px, painter %vx%v dp; want 80x60 px, 40x30 dp", d.Size, w, h)
	}
}

func TestArcFollowsCircle(t *testing.T) {
	var ops op.Ops
	p := &Painter{gtx: layout.Context{Ops: &ops}, scale: 1}
	for _, sweep := range []float32{90, -135, 360} {
		pa := p.Path()
		pa.ArcAround(50, 50, 40, 30, sweep)
		start := pa.segs[0].pts[0]
		if x, y := polar(50, 50, 40, 30); dist(start, f32.Pt(x, y)) > 0.01 {
			t.Errorf("sweep %v starts at %v, want (%v, %v)", sweep, start, x, y)
		}
		prev := start
		for _, s := range pa.segs[1:] {
			for _, tt := range []float32{0.25, 0.5, 0.75, 1} {
				pt := cubic(prev, s.pts[0], s.pts[1], s.pts[2], tt)
				if r := dist(pt, f32.Pt(50, 50)); math.Abs(float64(r-40)) > 0.05 {
					t.Fatalf("sweep %v: point %v is %v from the center, want 40", sweep, pt, r)
				}
			}
			prev = s.pts[2]
		}
		if x, y := polar(50, 50, 40, 30+sweep); dist(prev, f32.Pt(x, y)) > 0.01 {
			t.Errorf("sweep %v ends at %v, want (%v, %v)", sweep, prev, x, y)
		}
	}
	// 90° clockwise from 12 o'clock is 3 o'clock.
	if x, y := polar(0, 0, 10, 90); math.Abs(float64(x-10)) > 1e-4 || math.Abs(float64(y)) > 1e-4 {
		t.Errorf("polar 90° = (%v, %v), want (10, 0)", x, y)
	}
}

func cubic(p0, p1, p2, p3 f32.Point, t float32) f32.Point {
	u := 1 - t
	return p0.Mul(u * u * u).Add(p1.Mul(3 * u * u * t)).Add(p2.Mul(3 * u * t * t)).Add(p3.Mul(t * t * t))
}

func dist(a, b f32.Point) float32 {
	d := a.Sub(b)
	return float32(math.Hypot(float64(d.X), float64(d.Y)))
}

func TestCompactOutsideResponsive(t *testing.T) {
	defer SetWindowSize(windowSize)
	SetMetric(unit.Metric{PxPerDp: 2})
	defer SetMetric(unit.Metric{PxPerDp: 1})
	SetWindowSize(image.Pt(1000, 800)) // 500 dp
	if !Compact() {
		t.Error("a 500 dp window isn't compact")
	}
	SetWindowSize(image.Pt(1400, 800)) // 700 dp
	if Compact() {
		t.Error("a 700 dp window is compact")
	}
}
