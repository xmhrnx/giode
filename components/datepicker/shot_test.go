package datepicker

import (
	"image"
	"image/png"
	"os"
	"runtime"
	"testing"
	"time"

	"gioui.org/gpu/headless"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/datetime"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// TestShot writes PNGs of calendars to GIODE_SHOT_DIR, for looking at.
func TestShot(t *testing.T) {
	dir := os.Getenv("GIODE_SHOT_DIR")
	if dir == "" {
		t.Skip("set GIODE_SHOT_DIR")
	}
	if runtime.GOOS == "linux" && os.Getenv("EGL_PLATFORM") == "" {
		t.Setenv("EGL_PLATFORM", "surfaceless")
	}
	shot := func(name string, w, h int, rtl bool, el layout.Widget) {
		win, err := headless.NewWindow(w, h)
		if err != nil {
			t.Skipf("no offscreen renderer: %v", err)
		}
		defer win.Release()
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Constraints: layout.Constraints{Max: image.Pt(w, h)}, Metric: unit.Metric{PxPerDp: 2, PxPerSp: 2}}
		if rtl {
			gtx.Locale = system.Locale{Direction: system.RTL}
		}
		paint.Fill(gtx.Ops, properties.CalcColor("theme.background"))
		frame(gtx, rawElement(el))
		if err := win.Frame(&ops); err != nil {
			t.Fatal(err)
		}
		img := image.NewRGBA(image.Rectangle{Max: image.Pt(w, h)})
		if err := win.Screenshot(img); err != nil {
			t.Fatal(err)
		}
		f, _ := os.Create(dir + "/" + name + ".png")
		png.Encode(f, img)
		f.Close()
	}
	now := time.Now()
	c := NewCalendar().Locale(datetime.NewLocale("en-US"))
	c.SetSelected(now)
	shot("calendar-en", 560, 640, false, c.Layout)
	r := NewCalendar().Locale(datetime.NewLocale("en-GB")).Range(true).Months(2)
	r.SetRange(now, datetime.AddDays(now, 9))
	shot("calendar-range", 1100, 640, false, r.Layout)
	fa := NewCalendar().Locale(datetime.NewLocale("fa-IR"))
	fa.SetSelected(now)
	shot("calendar-fa", 560, 640, true, fa.Layout)
	dp := New(styles.Styles{Width: 220}).Locale(datetime.NewLocale("en-US")).Label("Due date")
	dp.SetValue(now)
	dp.openCalendar()
	shot("datepicker-open", 700, 820, false, func(gtx layout.Context) layout.Dimensions {
		return elements.Box(styles.Styles{Padding: properties.UniformInset(20)}, dp).Layout(gtx)
	})
	tp := NewTime(styles.Styles{Width: 160}).Locale(datetime.NewLocale("en-US"))
	tp.SetValue(TimeOfDay{14, 30})
	tp.openList()
	shot("timepicker-open", 500, 700, false, func(gtx layout.Context) layout.Dimensions {
		return elements.Box(styles.Styles{Padding: properties.UniformInset(20)}, tp).Layout(gtx)
	})
}
