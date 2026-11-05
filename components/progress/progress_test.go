package progress

import (
	"image"
	"os"
	"runtime"
	"testing"

	"gioui.org/gpu/headless"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/styles"
)

// render draws a 200x8 bar at value v and returns the fill's extent
// along its middle row: the pixels in the fill colour.
func render(t *testing.T, v float32, rtl bool) (first, last int) {
	if runtime.GOOS == "linux" && os.Getenv("EGL_PLATFORM") == "" {
		t.Setenv("EGL_PLATFORM", "surfaceless")
	}
	win, err := headless.NewWindow(200, 8)
	if err != nil {
		t.Skipf("no offscreen renderer: %v", err)
	}
	defer win.Release()
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: layout.Constraints{Max: image.Pt(200, 8)}}
	if rtl {
		gtx.Locale = system.Locale{Direction: system.RTL}
	}
	Progress(v, styles.Styles{Color: "#ff0000", Background: "#0000ff"}).Layout(gtx)
	if err := win.Frame(&ops); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 200, 8))
	if err := win.Screenshot(img); err != nil {
		t.Fatal(err)
	}
	first, last = -1, -1
	for x := 0; x < 200; x++ {
		c := img.RGBAAt(x, 4)
		if c.R > 200 && c.B < 50 {
			if first < 0 {
				first = x
			}
			last = x
		}
	}
	return first, last
}

func TestProgressFill(t *testing.T) {
	if f, l := render(t, 0.5, false); f > 1 || l < 97 || l > 102 {
		t.Errorf("half: fill %d..%d, want 0..~99", f, l)
	}
	if f, l := render(t, 0.5, true); f < 97 || f > 102 || l < 198 {
		t.Errorf("half, right to left: fill %d..%d, want ~100..199", f, l)
	}
	if f, _ := render(t, 0, false); f >= 0 {
		t.Errorf("empty: fill from %d", f)
	}
	if f, l := render(t, 1, false); f > 1 || l < 198 {
		t.Errorf("full: fill %d..%d", f, l)
	}
}
