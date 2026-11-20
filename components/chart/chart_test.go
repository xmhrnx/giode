package chart

import (
	"image"
	"math"
	"strings"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/fonts"
)

func frameView(gtx layout.Context, view elements.Element) {
	elements.BeginFrame()
	defer elements.EndFrame()
	elements.SetWindowSize(gtx.Constraints.Max)
	elements.SetMetric(gtx.Metric)
	view.Layout(gtx)
	elements.DrainContextMenus(gtx)
	elements.DrainPortals(gtx)
	elements.DispatchBack(gtx)
	elements.DispatchKeys(gtx)
	elements.RunAfterLayout()
}

type harness struct {
	r   input.Router
	c   *Chart
	now time.Time
}

func newHarness(c *Chart) *harness {
	fonts.UseGofont()
	h := &harness{c: c, now: time.Unix(1000, 0)}
	h.frame()
	h.frame()
	return h
}

func (h *harness) frame() {
	h.now = h.now.Add(500 * time.Millisecond)
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Now: h.now, Source: h.r.Source(), Constraints: layout.Constraints{Max: image.Pt(600, 400)},
		Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
	frameView(gtx, h.c)
	h.r.Frame(&ops)
}

func (h *harness) move(p f32.Point) {
	h.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: p})
	h.frame()
}

func (h *harness) click(p f32.Point) {
	h.r.Queue(
		pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: p},
		pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: p, Buttons: pointer.ButtonPrimary},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: p},
	)
	h.frame()
}

func TestNiceTicks(t *testing.T) {
	ticks, lo, hi := niceTicks(3, 97, 6)
	if lo != 0 || hi != 100 || len(ticks) != 6 || ticks[1] != 20 {
		t.Errorf("ticks %v %v %v", ticks, lo, hi)
	}
	ticks, lo, hi = niceTicks(-0.37, 0.81, 5)
	if lo > -0.37 || hi < 0.81 || len(ticks) < 4 {
		t.Errorf("ticks %v %v %v", ticks, lo, hi)
	}
	for in, want := range map[float64]string{0: "0", 950: "950", 12345: "12.3k", 3.4e6: "3.4M", 2e9: "2B", 1.25: "1.25", -1500.0: "-1500", 0.001234: "0.0012"} {
		if got := FormatNumber(in); got != want {
			t.Errorf("FormatNumber(%v) = %q, want %q", in, got, want)
		}
	}
	day := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	xt, u := timeTicks(TimeX(day), TimeX(day.AddDate(0, 0, 30)), 6, time.UTC)
	if u.d != 7*24*time.Hour || len(xt) < 4 || time.Unix(int64(xt[0]), 0).UTC().Weekday() != time.Monday {
		t.Errorf("month: %v ticks every %v", len(xt), u.d)
	}
	xt, u = timeTicks(TimeX(day), TimeX(day.AddDate(3, 0, 0)), 6, time.UTC)
	if u.months != 6 || time.Unix(int64(xt[0]), 0).UTC().Day() != 1 {
		t.Errorf("years: every %d months from %v", u.months, time.Unix(int64(xt[0]), 0).UTC())
	}
}

func TestDownsample(t *testing.T) {
	var pts []Point
	for i := 0; i < 10000; i++ {
		y := math.Sin(float64(i) / 300)
		if i == 4321 {
			y = 50 // a spike must survive
		}
		pts = append(pts, Point{X: float64(i), Y: y})
	}
	out := downsample(pts, 300)
	if len(out) != 300 || out[0] != pts[0] || out[299] != pts[9999] {
		t.Fatalf("len %d", len(out))
	}
	spike := false
	for _, p := range out {
		if p.Y == 50 {
			spike = true
		}
	}
	if !spike {
		t.Error("the spike was dropped")
	}
}

func TestBarHoverSelectLegend(t *testing.T) {
	var sel [2]int
	c := New(Bar).Categories("Q1", "Q2", "Q3", "Q4").SetSeries(
		Series{Name: "2025", Values: []float64{10, 20, 30, 40}},
		Series{Name: "2026", Values: []float64{15, 25, 35, 45}},
	).OnSelect(func(s, i int) { sel = [2]int{s, i} }).Animate(false)
	h := newHarness(c)
	f := c.frame
	if f == nil || len(f.hits) != 8 {
		t.Fatalf("hits %v", f)
	}
	// The second series' third bar.
	var bar hitRect
	for _, hr := range f.hits {
		if hr.series == 1 && hr.index == 2 {
			bar = hr
		}
	}
	center := f32.Pt(float32(bar.r.Min.X+bar.r.Dx()/2), float32(bar.r.Max.Y-2))
	h.move(center)
	head, rows := c.tooltip()
	if head != "Q3" || len(rows) != 2 || rows[1].value != "35" {
		t.Errorf("tooltip %q %+v", head, rows)
	}
	h.click(center)
	if sel != [2]int{1, 2} {
		t.Errorf("selected %v", sel)
	}
	// The legend hides a series.
	lh := c.legendAt[0].r
	h.click(f32.Pt(float32(lh.Min.X+2), float32(lh.Min.Y+lh.Dy()/2)))
	if !c.series[0].Hidden {
		t.Error("legend didn't hide the series")
	}
	h.frame()
	if n := len(c.frame.hits); n != 4 {
		t.Errorf("%d bars after hiding", n)
	}
	if s := c.Summary(); !strings.Contains(s, "Bar chart") || !strings.Contains(s, "2026: from 15 to 45") {
		t.Errorf("summary %q", s)
	}
}

func TestTimeLineBrushAndZoom(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var pts []Point
	for i := 0; i < 5000; i++ {
		pts = append(pts, Point{X: TimeX(start.Add(time.Duration(i) * time.Hour)), Y: float64(i % 100)})
	}
	var from, to float64
	c := New(Line).X(Axis{Time: true, Location: time.UTC}).SetSeries(Series{Name: "load", Points: pts}).
		OnBrush(func(a, b float64) { from, to = a, b }).Zoomable(true).Animate(false)
	h := newHarness(c)
	f := c.frame
	if f.categorical || f.plot.Empty() {
		t.Fatalf("frame %+v", f)
	}
	a := f32.Pt(float32(f.plot.Min.X+100), float32(f.plot.Min.Y+20))
	b := f32.Pt(float32(f.plot.Min.X+300), float32(f.plot.Min.Y+20))
	h.r.Queue(
		pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: a, Buttons: pointer.ButtonPrimary},
		pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: b, Buttons: pointer.ButtonPrimary},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: b},
	)
	h.frame()
	if !(from > pts[0].X && to > from && to < pts[len(pts)-1].X) {
		t.Errorf("brushed %v to %v", from, to)
	}
	// The wheel zooms in around the pointer.
	h.r.Queue(pointer.Event{Kind: pointer.Scroll, Source: pointer.Mouse, Position: b, Scroll: f32.Pt(0, -200)})
	h.frame()
	h.frame()
	if !c.view.set || c.view.hi-c.view.lo >= c.view.dhi-c.view.dlo {
		t.Errorf("view %+v", c.view)
	}
	h.move(f32.Pt(float32(f.plot.Min.X+200), float32(f.plot.Min.Y+20)))
	if head, rows := c.tooltip(); head == "" || len(rows) != 1 {
		t.Errorf("tooltip %q %v", head, rows)
	}
}

func TestPieAndAnimation(t *testing.T) {
	c := New(Donut).Categories("A", "B", "C").SetSeries(Series{Name: "Share", Values: []float64{50, 30, 20}})
	h := newHarness(c)
	g := c.pie
	if len(g.starts) != 3 {
		t.Fatalf("slices %+v", g)
	}
	// The first slice starts at the top: a point to its right, inside
	// the ring, is in it.
	mid := (g.r0 + g.r1) / 2
	p := g.center.Add(f32.Pt(mid*0.5, -mid*0.85))
	h.move(p)
	if !c.hover.active || c.hover.index != 0 {
		t.Errorf("hover %+v", c.hover)
	}
	if head, rows := c.tooltip(); head != "A" || !strings.Contains(rows[0].value, "50%") {
		t.Errorf("tooltip %q %+v", head, rows)
	}
	// New data animates from the old.
	c.SetSeries(Series{Name: "Share", Values: []float64{10, 10, 80}})
	if !c.animating {
		t.Fatal("not animating")
	}
	h.frame()
	h.frame()
	if c.animating {
		t.Error("animation didn't finish")
	}
}

func TestSummaryAndTable(t *testing.T) {
	c := New(Line).Title("Revenue").Categories("Jan", "Feb").SetSeries(Series{Name: "EU", Values: []float64{1200, 3400}})
	if s := c.Summary(); s != "Revenue. Line chart. EU: from 1200 to 3400, low 1200, high 3400" {
		t.Errorf("summary %q", s)
	}
	if c.Table() == nil {
		t.Error("no table")
	}
	g := New(Gauge).SetGauge(72, 0, 100)
	if s := g.Summary(); s != "Gauge: 72 of 0 to 100" {
		t.Errorf("gauge %q", s)
	}
}
