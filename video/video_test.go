package video

import (
	"image"
	"image/color"
	"os"
	"runtime"
	"testing"
	"time"

	"gioui.org/gpu/headless"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

var (
	red  = color.RGBA{R: 255, A: 255}
	blue = color.RGBA{B: 255, A: 255}
)

// halves is a w×h frame, red on the left and blue on the right.
func halves(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x < w/2 {
				img.SetRGBA(x, y, red)
			} else {
				img.SetRGBA(x, y, blue)
			}
		}
	}
	return img
}

func render(t *testing.T, s *Surface, opts ViewOptions, w, h int) *image.RGBA {
	t.Helper()
	if runtime.GOOS == "linux" && os.Getenv("EGL_PLATFORM") == "" {
		t.Setenv("EGL_PLATFORM", "surfaceless")
	}
	win, err := headless.NewWindow(w, h)
	if err != nil {
		t.Skipf("no offscreen renderer: %v", err)
	}
	defer win.Release()
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Now: time.Now(), Constraints: layout.Exact(image.Pt(w, h)), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
	s.View(opts).Layout(gtx)
	if err := win.Frame(&ops); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	if err := win.Screenshot(img); err != nil {
		t.Fatal(err)
	}
	return img
}

func near(a, b color.RGBA) bool {
	d := func(x, y uint8) bool { return int(x)-int(y) < 40 && int(y)-int(x) < 40 }
	return d(a.R, b.R) && d(a.G, b.G) && d(a.B, b.B)
}

func TestViewPlacement(t *testing.T) {
	black := color.RGBA{A: 255}
	type probe struct {
		x, y int
		c    color.RGBA
	}
	cases := []struct {
		name   string
		opts   ViewOptions
		probes []probe
	}{
		{"contain", ViewOptions{}, []probe{{10, 40, red}, {70, 40, blue}, {40, 5, black}, {40, 75, black}}},
		{"mirror", ViewOptions{Mirror: true}, []probe{{10, 40, blue}, {70, 40, red}}},
		{"cover", ViewOptions{Fit: Cover}, []probe{{10, 5, red}, {70, 75, blue}}},
		{"stretch", ViewOptions{Fit: Stretch}, []probe{{10, 5, red}, {70, 75, blue}}},
		// Turned clockwise, the left half is on top.
		{"rotate90", ViewOptions{Rotation: 90}, []probe{{40, 10, red}, {40, 70, blue}, {5, 40, black}}},
		{"rotate270", ViewOptions{Rotation: 270}, []probe{{40, 10, blue}, {40, 70, red}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewSurface(nil)
			s.Push(halves(40, 20))
			img := render(t, s, tc.opts, 80, 80)
			for _, p := range tc.probes {
				if got := img.RGBAAt(p.x, p.y); !near(got, p.c) {
					t.Errorf("(%d, %d) = %v, want %v", p.x, p.y, got, p.c)
				}
			}
		})
	}
}

func TestPlaceholder(t *testing.T) {
	s := NewSurface(nil)
	img := render(t, s, ViewOptions{Background: color.NRGBA{G: 255, A: 255}}, 20, 20)
	if got := img.RGBAAt(10, 10); !near(got, color.RGBA{G: 255, A: 255}) {
		t.Errorf("empty view = %v", got)
	}
}

func TestBuffersAndOps(t *testing.T) {
	s := NewSurface(nil)
	s.Push(halves(4, 2))
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: layout.Exact(image.Pt(10, 10))}
	s.layout(gtx, ViewOptions{})
	shown := s.bufs[s.showing]
	first := s.op
	// A redraw without a new frame reuses the op: no new upload.
	s.layout(gtx, ViewOptions{})
	if s.op != first {
		t.Error("redraw made a new image op")
	}
	// New frames never land in the buffer being shown.
	for i := 0; i < 5; i++ {
		s.Push(halves(4, 2))
		if s.bufs[s.latest] == shown {
			t.Fatal("a frame was written into the buffer on screen")
		}
	}
	s.layout(gtx, ViewOptions{})
	if s.op == first {
		t.Error("a new frame kept the old op")
	}
	if s.Frames() != 6 || s.Size() != image.Pt(4, 2) {
		t.Errorf("frames %d, size %v", s.Frames(), s.Size())
	}
	if fps := s.FPS(); fps <= 0 {
		t.Errorf("fps = %v", fps)
	}
	s.Clear()
	if s.Size() != (image.Point{}) {
		t.Error("Clear kept the frame")
	}
	// Push copies: changing the source after doesn't change the frame.
	src := halves(2, 1)
	s.Push(src)
	src.SetRGBA(0, 0, blue)
	if got := s.bufs[s.latest].RGBAAt(0, 0); got != red {
		t.Errorf("frame changed with its source: %v", got)
	}
	// Other image types are converted.
	g := image.NewGray(image.Rect(0, 0, 2, 2))
	g.Pix[0] = 200
	s.Push(g)
	if got := s.bufs[s.latest].RGBAAt(0, 0); got != (color.RGBA{200, 200, 200, 255}) {
		t.Errorf("gray = %v", got)
	}
}

func TestI420(t *testing.T) {
	// 4×2: limited-range BT.601 red (Y 81, U 90, V 240) on the left,
	// white (Y 235) on the right.
	f := I420{
		Y:     []byte{81, 81, 235, 235, 81, 81, 235, 235},
		U:     []byte{90, 128},
		V:     []byte{240, 128},
		Width: 4, Height: 2, StrideY: 4, StrideU: 2, StrideV: 2,
	}
	s := NewSurface(nil)
	s.PushI420(f)
	img := s.bufs[s.latest]
	if got := img.RGBAAt(0, 1); got.R < 240 || got.G > 15 || got.B > 15 {
		t.Errorf("red = %v", got)
	}
	if got := img.RGBAAt(3, 0); got.R < 250 || got.G < 250 || got.B < 250 {
		t.Errorf("white = %v", got)
	}
}
