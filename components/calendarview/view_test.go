package calendarview

import (
	"image"
	"image/png"
	"os"
	"runtime"
	"testing"
	"time"

	"gioui.org/gpu/headless"
	"gioui.org/io/input"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/datetime"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles/properties"
)

// frame runs a frame's steps as ui.Frame does.
func frame(gtx layout.Context, view layout.Widget) {
	elements.BeginFrame()
	defer elements.EndFrame()
	elements.SetWindowSize(gtx.Constraints.Max)
	elements.SetMetric(gtx.Metric)
	view(gtx)
	elements.DrainContextMenus(gtx)
	elements.DrainPortals(gtx)
	elements.DispatchBack(gtx)
	elements.DispatchKeys(gtx)
	elements.RunAfterLayout()
}

func sample() []Event {
	d := func(day, h, m int) time.Time { return time.Date(2026, 11, day, h, m, 0, 0, time.UTC) }
	return []Event{
		{ID: "1", Title: "Standup", Start: d(16, 9, 30), End: d(16, 9, 45)},
		{ID: "2", Title: "Design review", Start: d(17, 10, 0), End: d(17, 11, 30), Color: "#8b5cf6"},
		{ID: "3", Title: "1:1 with Sam", Start: d(17, 10, 30), End: d(17, 11, 0), Color: "#16a34a"},
		{ID: "4", Title: "Lunch", Start: d(18, 12, 0), End: d(18, 13, 0), Color: "#d97706"},
		{ID: "5", Title: "Offsite", AllDay: true, Start: d(18, 0, 0), End: d(21, 0, 0), Color: "#0284c7"},
		{ID: "6", Title: "Release", Start: d(20, 15, 0), End: d(20, 17, 0), Color: "#dc2626"},
		{ID: "7", Title: "Planning", Start: d(20, 9, 0), End: d(20, 10, 0)},
		{ID: "8", Title: "Retro", Start: d(20, 11, 0), End: d(20, 12, 0)},
		{ID: "9", Title: "Interview", Start: d(20, 13, 0), End: d(20, 14, 0)},
		{ID: "10", Title: "Overnight deploy", Start: d(19, 22, 0), End: d(20, 2, 0), Color: "#64748b"},
	}
}

func fixedNow() time.Time { return time.Date(2026, 11, 20, 14, 20, 0, 0, time.UTC) }

func TestShot(t *testing.T) {
	dir := os.Getenv("GIODE_SHOT_DIR")
	if dir == "" {
		t.Skip("set GIODE_SHOT_DIR")
	}
	if runtime.GOOS == "linux" && os.Getenv("EGL_PLATFORM") == "" {
		t.Setenv("EGL_PLATFORM", "surfaceless")
	}
	shot := func(name string, w, h int, rtl bool, v *CalendarView) {
		win, err := headless.NewWindow(w, h)
		if err != nil {
			t.Skipf("no offscreen renderer: %v", err)
		}
		defer win.Release()
		var r input.Router
		for range 2 {
			var ops op.Ops
			gtx := layout.Context{Ops: &ops, Source: r.Source(), Constraints: layout.Exact(image.Pt(w, h)), Metric: unit.Metric{PxPerDp: 2, PxPerSp: 2}}
			if rtl {
				gtx.Locale = system.Locale{Direction: system.RTL}
			}
			paint.Fill(gtx.Ops, properties.CalcColor("theme.background"))
			frame(gtx, v.Layout)
			r.Frame(&ops)
			if err := win.Frame(&ops); err != nil {
				t.Fatal(err)
			}
		}
		img := image.NewRGBA(image.Rectangle{Max: image.Pt(w, h)})
		if err := win.Screenshot(img); err != nil {
			t.Fatal(err)
		}
		f, _ := os.Create(dir + "/" + name + ".png")
		png.Encode(f, img)
		f.Close()
	}
	mk := func(m Mode, tag string) *CalendarView {
		v := New().Locale(datetime.NewLocale(tag)).Zone(time.UTC).SetEvents(sample()).Mode(m)
		v.now = fixedNow
		v.Show(fixedNow())
		return v
	}
	shot("cal-week", 1800, 1400, false, mk(Week, "en-US").SecondZone(mustZone("Asia/Tehran")))
	shot("cal-month", 1800, 1400, false, mk(Month, "en-US"))
	shot("cal-day", 900, 1400, false, mk(Day, "en-GB"))
	shot("cal-agenda", 1200, 1000, false, mk(Agenda, "en-US").AgendaDays(7))
	shot("cal-month-fa", 1800, 1400, true, mk(Month, "fa-IR"))
}

func mustZone(name string) *time.Location {
	z, _ := time.LoadLocation(name)
	return z
}
