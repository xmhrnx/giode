package dialog

import (
	"image"
	"image/color"
	"os"
	"runtime"
	"testing"

	"gioui.org/gpu/headless"
	"gioui.org/io/input"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// The panel's shadow falls outside it; it used to be clipped to the
// panel's rectangle, leaving dark squares under the rounded corners.
func TestShadowNotClipped(t *testing.T) {
	if runtime.GOOS == "linux" && os.Getenv("EGL_PLATFORM") == "" {
		t.Setenv("EGL_PLATFORM", "surfaceless")
	}
	win, err := headless.NewWindow(600, 400)
	if err != nil {
		t.Skipf("no offscreen renderer: %v", err)
	}
	defer win.Release()
	open := true
	d := New(&open, func() elements.Element { return elements.Box(styles.Styles{Width: 300, Height: 200}) },
		styles.Styles{Background: "#18181b", BorderRadius: 16, Padding: properties.UniformInset(0),
			BoxShadow: properties.BoxShadow{Y: 18, Blur: 48, Color: "#000000e6"}})
	var r input.Router
	for range 3 {
		var ops op.Ops
		paint.Fill(&ops, color.NRGBA{200, 200, 200, 255})
		gtx := layout.Context{Ops: &ops, Source: r.Source(), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(600, 400))}
		elements.SetWindowSize(image.Pt(600, 400))
		elements.BeginFrame()
		d.Layout(gtx)
		elements.DrainPortals(gtx)
		r.Frame(&ops)
		if err := win.Frame(&ops); err != nil {
			t.Fatal(err)
		}
	}
	img := image.NewRGBA(image.Rect(0, 0, 600, 400))
	if err := win.Screenshot(img); err != nil {
		t.Fatal(err)
	}
	// The panel is 340x240 (an all-zero Padding means the default 20),
	// centered: x 130-470, y 80-320.
	luma := func(x, y int) int { c := img.RGBAAt(x, y); return int(c.R) + int(c.G) + int(c.B) }
	scrim := luma(5, 5)
	if beside := luma(120, 200); beside >= scrim {
		t.Errorf("no shadow beside the panel (%d, scrim %d): it's clipped to the panel", beside, scrim)
	}
	// Under the top-left rounded corner, the shadow continues smoothly
	// from just outside the panel; a clipped one steps at the edge.
	if in, out := luma(131, 81), luma(128, 81); in-out > 12 || out-in > 12 {
		t.Errorf("the shadow steps at the panel's edge: %d inside the corner, %d just outside", in, out)
	}
}
