package colorpicker

import (
	"image"
	"image/color"
	"math"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/fonts"
)

func TestHSVRoundTrip(t *testing.T) {
	for _, c := range []color.NRGBA{
		{255, 0, 0, 255}, {0, 128, 255, 255}, {51, 102, 255, 128}, {12, 34, 56, 255}, {200, 200, 200, 255}, {0, 0, 0, 255},
	} {
		if got := fromRGB(c, 0).rgb(); got != c {
			t.Errorf("%v → HSV → %v", c, got)
		}
	}
	// A grey keeps the hue it had.
	if h := fromRGB(color.NRGBA{128, 128, 128, 255}, 210).h; h != 210 {
		t.Errorf("grey's hue %v, want the kept 210", h)
	}
}

type harness struct {
	r input.Router
	p *ColorPicker
}

func (h *harness) frame() layout.Dimensions {
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Source: h.r.Source(), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Constraints{Max: image.Pt(400, 600)}}
	dims := h.p.Layout(gtx)
	h.r.Frame(&ops)
	return dims
}

func (h *harness) click(x, y float32) {
	p := f32.Pt(x, y)
	h.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: p, Buttons: pointer.ButtonPrimary},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: p})
	h.frame()
	h.frame()
}

func TestColorPicker(t *testing.T) {
	fonts.UseGofont()
	var got []string
	p := New("#FF0000").Presets("#00FF00", "#FFFFFF", "theme.primary").OnChange(func(c string) { got = append(got, c) })
	h := &harness{p: p}
	dims := h.frame()
	if dims.Size.X != 240 {
		t.Errorf("width %d, want the default 240", dims.Size.X)
	}
	if p.Value() != "#FF0000" || p.Field().Text() != "#FF0000" {
		t.Fatalf("value %q, field %q; want #FF0000", p.Value(), p.Field().Text())
	}

	// The square: across is saturation, down is brightness.
	h.click(120, 40) // s 0.5, v 0.75
	c := p.c
	if math.Abs(c.s-0.5) > 0.01 || math.Abs(c.v-0.75) > 0.01 || c.h != 0 {
		t.Errorf("click in the square: %+v, want s 0.5, v 0.75, hue kept", c)
	}
	if want := "#BF6060"; p.Value() != want || p.Field().Text() != want {
		t.Errorf("value %q, field %q; want %q", p.Value(), p.Field().Text(), want)
	}

	// The hue bar, under the 160px square and a 10px gap: 170 to 184.
	h.click(120, 177)
	if math.Abs(p.c.h-180) > 1 {
		t.Errorf("click mid hue bar: hue %v, want 180", p.c.h)
	}

	// Typing a hex applies it as soon as it reads as one.
	p.Field().Focus()
	h.frame()
	p.Field().SetText("")
	h.r.Queue(key.EditEvent{Text: "3366ff"})
	h.frame()
	if p.Value() != "#3366FF" {
		t.Errorf("typed 3366ff: %q", p.Value())
	}

	// A preset, under the field (at 160+10+14+10, about 35 tall) and a gap.
	y := float32(160 + 10 + 14 + 10 + h.fieldHeight() + 10 + 11)
	h.click(11, y)
	if p.Value() != "#00FF00" {
		t.Errorf("clicked the first preset: %q, want #00FF00", p.Value())
	}
	if len(got) < 4 || got[len(got)-1] != "#00FF00" {
		t.Errorf("OnChange got %q", got)
	}

	// Without Alpha a color is opaque; with it, the bar sets alpha and
	// the value has eight digits.
	p.SetValue("#11223380")
	if p.Value() != "#112233" {
		t.Errorf("without Alpha: %q, want opaque #112233", p.Value())
	}
	p.Alpha(true)
	h.frame()
	h.click(60, 160+10+14+10+7) // a quarter along the alpha bar
	if a := p.NRGBA().A; a < 60 || a > 68 {
		t.Errorf("alpha %d, want about a quarter", a)
	}
	if v := p.Value(); len(v) != 9 {
		t.Errorf("transparent value %q, want #RRGGBBAA", v)
	}
}

func (h *harness) fieldHeight() int {
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Constraints{Max: image.Pt(240, 600)}}
	return h.p.hex.Layout(gtx).Size.Y
}
