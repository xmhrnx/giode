package toaster

import (
	"image"
	"image/png"
	"os"
	"runtime"
	"testing"

	"gioui.org/gpu/headless"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/styles/properties"
)

// The accent bar is a short bar inset from the start edge, not a strip
// along the rounded edge.
func TestStripeInset(t *testing.T) {
	if runtime.GOOS == "linux" && os.Getenv("EGL_PLATFORM") == "" {
		t.Setenv("EGL_PLATFORM", "surfaceless")
	}
	const w, h = 300, 60
	win, err := headless.NewWindow(w, h)
	if err != nil {
		t.Skipf("no offscreen renderer: %v", err)
	}
	defer win.Release()
	tt := New()
	tt.Show(Error, "Something went wrong")
	tt.sync()
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: layout.Exact(image.Pt(w, h))}
	tt.toastView(tt.toasts[0]).Layout(gtx)
	if err := win.Frame(&ops); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	if err := win.Screenshot(img); err != nil {
		t.Fatal(err)
	}
	if out := os.Getenv("TOAST_PNG"); out != "" {
		f, _ := os.Create(out)
		png.Encode(f, img)
		f.Close()
	}
	want := properties.CalcColor(Error.color())
	is := func(x, y int) bool {
		c := img.RGBAAt(x, y)
		d := func(a, b uint8) int { return max(int(a), int(b)) - min(int(a), int(b)) }
		return d(c.R, want.R) < 40 && d(c.G, want.G) < 40 && d(c.B, want.B) < 40
	}
	if !is(stripeInset+1, h/2) {
		t.Errorf("no bar at x=%d: %v", stripeInset+1, img.RGBAAt(stripeInset+1, h/2))
	}
	for _, p := range []image.Point{{1, h / 2}, {stripeInset + 1, 2}, {stripeInset + 1, h - 3}} {
		if is(p.X, p.Y) {
			t.Errorf("bar color at %v: the bar should be inset", p)
		}
	}
}
