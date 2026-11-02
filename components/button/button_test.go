package button

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

type harness struct {
	r input.Router
	b *Button
}

func (h *harness) frame() layout.Dimensions {
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: layout.Constraints{Max: image.Pt(300, 100)}, Source: h.r.Source()}
	d := h.b.Layout(gtx)
	h.r.Frame(&ops)
	return d
}

func (h *harness) click() {
	pos := f32.Pt(5, 5)
	h.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: pos, Buttons: pointer.ButtonPrimary})
	h.frame()
	h.r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: pos})
	h.frame()
}

func TestDisabledAndLoadingIgnoreClicks(t *testing.T) {
	clicks := 0
	h := &harness{b: New("Save").OnClick(func() { clicks++ })}
	h.frame()
	h.click()
	if clicks != 1 {
		t.Fatalf("enabled button: %d clicks", clicks)
	}
	h.b.Disabled(true)
	h.click()
	h.b.Click()
	h.frame()
	h.b.Disabled(false).Loading(true)
	h.click()
	if clicks != 1 {
		t.Errorf("disabled/loading button clicked: %d clicks", clicks)
	}
	h.b.Loading(false)
	h.frame() // the re-enabled button registers its hit area again
	h.click()
	if clicks != 2 {
		t.Errorf("re-enabled button: %d clicks, want 2", clicks)
	}
}

func TestIconsFollowLabelColor(t *testing.T) {
	var seen []string
	probe := elements.Raw(func(gtx layout.Context) layout.Dimensions {
		seen = append(seen, elements.CurrentColor())
		return layout.Dimensions{Size: image.Pt(10, 10)}
	})
	h := &harness{b: New("Next", styles.Styles{Color: "#ff0000"}).Icon(probe).TrailingIcon(probe)}
	h.frame()
	if len(seen) != 2 || seen[0] != "#ff0000" || seen[1] != "#ff0000" {
		t.Errorf("icons saw inherited colors %v, want the label color twice", seen)
	}
}

func TestLoadingReplacesIcon(t *testing.T) {
	built := 0
	icon := elements.Raw(func(gtx layout.Context) layout.Dimensions { built++; return layout.Dimensions{} })
	h := &harness{b: New("Save").Icon(icon).Loading(true)}
	h.frame()
	if built != 0 {
		t.Error("icon laid out while loading")
	}
}

func TestShadeGradient(t *testing.T) {
	g := properties.Linear(90, "#808080", "#404040")
	got := shadeGradient(g, kit.Hovered)
	if got == g || got.Angle != 90 || len(got.Stops) != 2 {
		t.Fatalf("shadeGradient = %+v", got)
	}
	if properties.CalcColor(got.Stops[0].Color).R <= 0x80 {
		t.Error("hover shade didn't lighten the first stop")
	}
	if g.Stops[0].Color != "#808080" {
		t.Error("original gradient modified")
	}
}

func TestTouchTargetGrowsHitArea(t *testing.T) {
	defer elements.SetTouchInput(elements.TouchInput())
	clicks := 0
	h := &harness{b: New("OK", styles.Styles{Padding: properties.UniformInset(2)}).OnClick(func() { clicks++ })}

	elements.SetTouchInput(false)
	mouse := h.frame().Size
	if mouse.Y >= 48 {
		t.Fatalf("mouse: button already %v tall; the test needs a short one", mouse)
	}

	elements.SetTouchInput(true)
	touch := h.frame().Size
	if touch.Y != 48 || touch.X < 48 {
		t.Fatalf("touch: size %v, want at least 48x48", touch)
	}
	// A tap in the extra room above the visual still clicks.
	pos := f32.Pt(float32(touch.X)/2, 1)
	h.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Touch, Position: pos, Buttons: pointer.ButtonPrimary},
		pointer.Event{Kind: pointer.Release, Source: pointer.Touch, Position: pos})
	h.frame()
	if clicks != 1 {
		t.Errorf("tap in the touch margin: %d clicks, want 1", clicks)
	}
}
