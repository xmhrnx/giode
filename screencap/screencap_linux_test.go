//go:build linux && !android

package screencap

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

// TestCaptureX11 paints the root window's corner red and captures it.
// It runs where an X server is (Xvfb in a container); CI's has none.
func TestCaptureX11(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("no X display")
	}
	c, err := xgb.NewConn()
	if err != nil {
		t.Skip(err)
	}
	defer c.Close()
	screen := xproto.Setup(c).DefaultScreen(c)
	gc, _ := xproto.NewGcontextId(c)
	xproto.CreateGC(c, gc, xproto.Drawable(screen.Root), xproto.GcForeground, []uint32{0xff0000})
	xproto.PolyFillRectangle(c, xproto.Drawable(screen.Root), gc, []xproto.Rectangle{{X: 0, Y: 0, Width: 200, Height: 100}})
	c.Sync()

	srcs, err := Sources(context.Background())
	if err != nil || len(srcs) == 0 {
		t.Fatalf("sources %v %v", srcs, err)
	}
	t.Logf("sources %+v", srcs)
	frames := make(chan Frame, 1)
	cap, err := Start(context.Background(), nil, Options{FPS: 20, MaxSize: -1}, func(f Frame) {
		select {
		case frames <- f:
		default:
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer cap.Stop()
	select {
	case f := <-frames:
		px := f.Image.RGBAAt(10, 10)
		if px.R != 255 || px.G != 0 || px.B != 0 {
			t.Errorf("corner %v", px)
		}
		if f.Image.Bounds() != srcs[0].Bounds.Sub(srcs[0].Bounds.Min) {
			t.Errorf("size %v, display %v", f.Image.Bounds(), srcs[0].Bounds)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no frame")
	}
}

func TestScaledSize(t *testing.T) {
	c := &Capture{opts: Options{MaxSize: 1920}}
	if w, h := c.scaledSize(3840, 2160); w != 1920 || h != 1080 {
		t.Errorf("%d %d", w, h)
	}
	if w, h := c.scaledSize(1080, 1920); w != 1080 || h != 1920 {
		t.Errorf("%d %d", w, h)
	}
}
