// Package screencap captures the screen, for sharing it in a call or
// recording it: a stream of frames from a display.
//
//	srcs, _ := screencap.Sources(ctx)
//	c, err := screencap.Start(ctx, app, screencap.Options{Source: srcs[0].ID, FPS: 15},
//		func(f screencap.Frame) { encoder.Encode(f.Image) })
//	...
//	c.Stop()
//
// Where it runs: Windows (GDI), macOS (ScreenCaptureKit, macOS 12.3 and
// later; the user allows screen recording in System Settings), Linux on
// X11 (on Wayland, through Xwayland, only X11 windows show), Android
// (MediaProjection: the user confirms each time) and iOS (ReplayKit:
// the app's own screen). Elsewhere Start returns giode.ErrNotSupported.
package screencap

import (
	"context"
	"errors"
	"image"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xmhrnx/giode"
)

// ErrPermissionDenied is returned when the user didn't allow screen
// capture.
var ErrPermissionDenied = errors.New("screencap: permission denied")

// Source is something to capture: a display.
type Source struct {
	ID   string
	Name string
	// Bounds are the display's place on the desktop, in pixels.
	Bounds image.Rectangle
	// Primary is the main display.
	Primary bool
}

// Options configure Start.
type Options struct {
	// Source is the ID of the display to capture; the primary one when
	// empty.
	Source string
	// FPS is how many frames a second to capture: 10 by default.
	FPS int
	// MaxSize caps the frames' long side, scaling them down: 1920 by
	// default; -1 for full size.
	MaxSize int
}

// Frame is a captured frame.
type Frame struct {
	// Image is the frame. It's onFrame's to keep.
	Image *image.RGBA
	Time  time.Time
}

// Capture is a running capture.
type Capture struct {
	opts    Options
	onFrame func(Frame)
	plat    platformCapture
	stopped atomic.Bool
	busy    atomic.Bool
	once    sync.Once
}

// Sources lists the displays.
func Sources(ctx context.Context) ([]Source, error) { return sources(ctx) }

// Start captures a display, calling onFrame with each frame on its own
// goroutine. Frames arriving while onFrame still runs are dropped, so a
// slow consumer never builds a backlog. On Android and macOS it asks
// the user first.
func Start(ctx context.Context, a *giode.App, opts Options, onFrame func(Frame)) (*Capture, error) {
	if opts.FPS <= 0 {
		opts.FPS = 10
	}
	if opts.MaxSize == 0 {
		opts.MaxSize = 1920
	}
	c := &Capture{opts: opts, onFrame: onFrame}
	if err := c.start(ctx, a); err != nil {
		return nil, err
	}
	return c, nil
}

// Stop ends the capture.
func (c *Capture) Stop() {
	c.once.Do(func() {
		c.stopped.Store(true)
		c.plat.stop()
	})
}

// deliver hands a frame to onFrame unless it's still busy.
func (c *Capture) deliver(img *image.RGBA) {
	if c.stopped.Load() || !c.busy.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer c.busy.Store(false)
		c.onFrame(Frame{Image: img, Time: time.Now()})
	}()
}

// scaledSize is a frame's size within MaxSize.
func (c *Capture) scaledSize(w, h int) (int, int) {
	m := c.opts.MaxSize
	if m <= 0 || max(w, h) <= m {
		return w, h
	}
	if w >= h {
		return m, max(h*m/w, 1)
	}
	return max(w*m/h, 1), m
}

// fromBGRA makes an RGBA frame from BGRA pixels, scaled to fit MaxSize
// by sampling.
func (c *Capture) fromBGRA(src []byte, w, h, stride int) *image.RGBA {
	dw, dh := c.scaledSize(w, h)
	img := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < dh; y++ {
		sy := y * h / dh
		row := src[sy*stride:]
		out := img.Pix[y*img.Stride:]
		for x := 0; x < dw; x++ {
			sx := (x * w / dw) * 4
			o := x * 4
			out[o], out[o+1], out[o+2], out[o+3] = row[sx+2], row[sx+1], row[sx], 255
		}
	}
	return img
}

// ticker runs grab at the capture's rate until stopped.
func (c *Capture) ticker(stop <-chan struct{}, grab func() error) {
	t := time.NewTicker(time.Second / time.Duration(c.opts.FPS))
	defer t.Stop()
	for {
		if err := grab(); err != nil {
			return
		}
		select {
		case <-stop:
			return
		case <-t.C:
		}
	}
}

// fromNV12 makes an RGBA frame from NV12 planes (BT.709, as screens
// are), scaled to fit MaxSize.
func (c *Capture) fromNV12(y []byte, ys int, uv []byte, uvs int, w, h int, full bool) *image.RGBA {
	dw, dh := c.scaledSize(w, h)
	img := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for oy := 0; oy < dh; oy++ {
		sy := oy * h / dh
		out := img.Pix[oy*img.Stride:]
		for ox := 0; ox < dw; ox++ {
			sx := ox * w / dw
			yy := int(y[sy*ys+sx])
			ci := (sy/2)*uvs + (sx/2)*2
			u, v := int(uv[ci])-128, int(uv[ci+1])-128
			if !full {
				yy = (yy - 16) * 298 >> 8
				u, v = u*292>>8, v*292>>8
			}
			r := yy + (403*v)>>8
			g := yy - (48*u+120*v)>>8
			b := yy + (475*u)>>8
			o := ox * 4
			out[o], out[o+1], out[o+2], out[o+3] = clamp8(r), clamp8(g), clamp8(b), 255
		}
	}
	return img
}

func clamp8(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

// fromRGBA copies RGBA pixels into a frame, making them opaque.
func fromRGBA(src []byte, w, h, stride int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		row := img.Pix[y*img.Stride : y*img.Stride+w*4]
		copy(row, src[y*stride:])
		for x := 3; x < len(row); x += 4 {
			row[x] = 255
		}
	}
	return img
}
