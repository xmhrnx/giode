package chart

import (
	"image"
	"image/png"
	"math"
	"os"
	"runtime"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/gpu/headless"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// TestShot writes PNGs of charts to GIODE_SHOT_DIR, for looking at.
func TestShot(t *testing.T) {
	dir := os.Getenv("GIODE_SHOT_DIR")
	if dir == "" {
		t.Skip("set GIODE_SHOT_DIR")
	}
	if runtime.GOOS == "linux" && os.Getenv("EGL_PLATFORM") == "" {
		t.Setenv("EGL_PLATFORM", "surfaceless")
	}
	fonts.UseGofont()
	shot := func(name string, w, h int, rtl bool, hover *f32.Point, el elements.Element) {
		win, err := headless.NewWindow(w, h)
		if err != nil {
			t.Skipf("no offscreen renderer: %v", err)
		}
		defer win.Release()
		var r input.Router
		var img *image.RGBA
		for i := 0; i < 4; i++ {
			if i == 2 && hover != nil {
				r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: *hover})
			}
			var ops op.Ops
			gtx := layout.Context{Ops: &ops, Now: time.Unix(int64(i), 0), Source: r.Source(), Constraints: layout.Constraints{Max: image.Pt(w, h)}, Metric: unit.Metric{PxPerDp: 2, PxPerSp: 2}}
			if rtl {
				gtx.Locale = system.Locale{Direction: system.RTL}
			}
			paint.Fill(gtx.Ops, properties.CalcColor("theme.background"))
			frameView(gtx, el)
			r.Frame(&ops)
			if err := win.Frame(&ops); err != nil {
				t.Fatal(err)
			}
			img = image.NewRGBA(image.Rectangle{Max: image.Pt(w, h)})
			if err := win.Screenshot(img); err != nil {
				t.Fatal(err)
			}
		}
		f, _ := os.Create(dir + "/chart-" + name + ".png")
		png.Encode(f, img)
		f.Close()
	}
	q := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun"}
	bar := New(Bar).Title("Revenue by region").Categories(q...).SetSeries(
		Series{Name: "EU", Values: []float64{120, 180, 150, 210, 260, 240}},
		Series{Name: "US", Values: []float64{200, 220, 190, 250, 280, 310}},
		Series{Name: "Target", Values: []float64{180, 200, 200, 230, 260, 290}, Kind: Line, Dashed: true},
	).Animate(false).Y(Axis{Format: func(v float64) string { return "$" + FormatNumber(v) + "k" }})
	hp := f32.Pt(560, 300)
	shot("bar", 900, 560, false, &hp, bar)
	stacked := New(Bar).Categories(q...).SetSeries(
		Series{Name: "New", Values: []float64{12, 18, 15, 21, 26, 24}, Stack: "a"},
		Series{Name: "Expansion", Values: []float64{8, 9, 12, 10, 14, 16}, Stack: "a"},
		Series{Name: "Churn", Values: []float64{-4, -6, -3, -5, -7, -4}, Stack: "a"},
	).Animate(false)
	shot("stacked", 900, 520, false, nil, stacked)
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var a, b []Point
	for i := 0; i < 2000; i++ {
		x := TimeX(start.Add(time.Duration(i) * 2 * time.Hour))
		a = append(a, Point{X: x, Y: 50 + 30*math.Sin(float64(i)/80) + float64(i%17)})
		b = append(b, Point{X: x, Y: 30 + 20*math.Cos(float64(i)/120)})
	}
	line := New(Area).Title("Load").X(Axis{Time: true, Location: time.UTC}).SetSeries(
		Series{Name: "API", Points: a}, Series{Name: "Workers", Points: b, Kind: Line, Axis: 1},
	).Y2(Axis{}).Smooth(true).Animate(false)
	hp2 := f32.Pt(900, 300)
	shot("timeline", 1200, 560, false, &hp2, line)
	pie := New(Donut).Title("Pipeline by stage").Categories("Lead", "Qualified", "Proposal", "Won").
		SetSeries(Series{Name: "Deals", Values: []float64{120, 80, 45, 30}}).Animate(false)
	shot("donut", 700, 560, false, nil, pie)
	fun := New(Funnel).Title("Signup funnel").Categories("Visited", "Signed up", "Activated", "Paid").
		SetSeries(Series{Name: "Users", Values: []float64{12000, 4300, 2100, 640}}).Animate(false)
	shot("funnel", 900, 420, false, nil, fun)
	g := New(Gauge).Title("CPU").SetGauge(72, 0, 100, Band{60, "theme.success"}, Band{85, "theme.warning"}, Band{100, "theme.danger"}).Animate(false)
	shot("gauge", 500, 400, false, nil, g)
	var heat [][]float64
	for r := 0; r < 7; r++ {
		var row []float64
		for c := 0; c < 24; c++ {
			row = append(row, math.Max(0, 10*math.Sin(float64(c)/4)+float64(r*3)))
		}
		heat = append(heat, row)
	}
	cols := make([]string, 24)
	for i := range cols {
		cols[i] = itoa(i)
	}
	hm := New(Heatmap).Title("Activity").SetHeatmap([]string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}, cols, heat)
	shot("heatmap", 1000, 420, false, nil, hm)
	var sc []Point
	for i := 0; i < 60; i++ {
		x := float64(i)
		sc = append(sc, Point{X: x, Y: x*1.5 + 20*math.Sin(x), Size: float64(i%7 + 1)})
	}
	scat := New(Scatter).Title("Deals: size against age").SetSeries(Series{Name: "Deals", Points: sc}).Animate(false)
	shot("scatter", 900, 500, false, nil, scat)
	hbar := New(HorizontalBar).Categories("Search", "Ads", "Referral", "Social").
		SetSeries(Series{Name: "Visits", Values: []float64{4200, 2600, 1800, 900}}).Animate(false)
	shot("hbar", 900, 400, false, nil, hbar)
	shot("bar-rtl", 900, 560, true, nil, bar)
	spark := Sparkline([]float64{3, 5, 4, 6, 8, 7, 9, 12, 10, 13}, styles.Styles{Width: 160, Height: 40})
	shot("sparkline", 340, 90, false, nil, spark)
}
