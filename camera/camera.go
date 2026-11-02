// Package camera shows a live camera preview inside a Giode view and
// takes full-resolution pictures: a document scanner's viewfinder, a
// QR scanner, a photo booth.
//
//	cam, err := camera.Open(ctx, app, camera.Options{
//		OnFrame: func(f camera.Frame) { quad = findPage(f.Image) },
//	})
//	...
//	view := cam.Preview(camera.PreviewOptions{
//		Fill:    true,
//		Overlay: func(gtx layout.Context, toView func(f32.Point) f32.Point) { drawQuad(gtx, quad, toView) },
//	})
//	...
//	jpeg, err := cam.Capture(ctx)
//
// Frames arrive upright, whichever way the phone is held. The preview
// is drawn by Giode itself, so anything can be drawn over it, and
// OnFrame receives small grey frames for analysis (edge detection,
// barcode reading) on its own goroutine.
//
// Android (Camera2) and iOS and macOS (AVFoundation) are supported;
// elsewhere Open returns giode.ErrNotSupported. The app needs the
// camera permission: on Android the package adds it to the manifest;
// on iOS set ios.usage.camera in giode.toml (NSCameraUsageDescription).
package camera

import (
	"context"
	"errors"
	"image"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"gioui.org/f32"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/xmhrnx/giode"
)

// ErrPermissionDenied is returned by Open when the user didn't allow
// the app to use the camera.
var ErrPermissionDenied = errors.New("camera: permission denied")

// ErrClosed is returned by a closed Camera's methods.
var ErrClosed = errors.New("camera: closed")

// Facing selects the camera.
type Facing int

const (
	// Back is the main camera, facing away from the user.
	Back Facing = iota
	// Front faces the user; its preview is mirrored, as a mirror is.
	Front
)

// Options configure Open.
type Options struct {
	// Facing selects the back (default) or front camera.
	Facing Facing
	// AnalysisSize is the long side, in pixels, of the frames OnFrame
	// receives: 640 by default.
	AnalysisSize int
	// AnalysisRate caps how many frames per second OnFrame receives: 15
	// by default. Frames arriving while OnFrame still runs are dropped,
	// so a slow analysis never builds a backlog.
	AnalysisRate int
	// OnFrame receives analysis frames on its own goroutine; nil for
	// none. Use App.Post to hand results to the UI.
	OnFrame func(Frame)
	// OnError receives errors the camera reports while running (another
	// app took it, the hardware failed), from any goroutine. They're
	// logged when it's nil.
	OnError func(error)
}

// Frame is an analysis frame.
type Frame struct {
	// Image is the frame's brightness, upright, AnalysisSize on its long
	// side. It belongs to the callback: keep it, it isn't reused.
	Image *image.Gray
	// Time is when the frame arrived.
	Time time.Time
}

// Camera is an open camera. Its methods may be called from any
// goroutine.
type Camera struct {
	app  *giode.App
	opts Options
	plat platformCamera

	closed atomic.Bool

	// The preview: three buffers, so the converter never writes the
	// one being drawn or the latest one waiting to be.
	mu      sync.Mutex
	bufs    [3]*image.RGBA
	latest  int // index of the newest complete frame, -1 for none
	showing int // index the UI drew last
	size    image.Point
	frames  uint64

	raw      chan rawFrame
	analysis atomic.Bool // OnFrame is running
	lastAn   time.Time
	torch    atomic.Bool
}

// rawFrame is a frame from the platform, already copied out of its
// buffers.
type rawFrame struct {
	// Android: YUV 4:2:0 planes.
	y, u, v               []byte
	w, h                  int
	yStride               int
	uvStride, uvPixStride int
	rotation              int // clockwise degrees to turn the frame upright
	mirror                bool
	// Apple: BGRA, already upright.
	bgra   []byte
	stride int
}

// Supported reports whether this platform has a camera Open can use.
func Supported() bool { return supported }

// RequestPermission asks the user to let the app use the camera, if it
// hasn't yet, and reports whether it may. Open asks too; call this to
// ask at a moment of your choosing.
func RequestPermission(ctx context.Context, a *giode.App) (bool, error) {
	if !supported {
		return false, giode.ErrNotSupported
	}
	return requestPermission(ctx, a)
}

// Open starts the camera, asking for permission first if needed. Close
// it when the view showing it goes away.
func Open(ctx context.Context, a *giode.App, opts Options) (*Camera, error) {
	if !supported {
		return nil, giode.ErrNotSupported
	}
	if opts.AnalysisSize == 0 {
		opts.AnalysisSize = 640
	}
	if opts.AnalysisRate <= 0 {
		opts.AnalysisRate = 15
	}
	ok, err := requestPermission(ctx, a)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrPermissionDenied
	}
	c := &Camera{app: a, opts: opts, latest: -1, showing: -1, raw: make(chan rawFrame, 1)}
	go c.convert()
	if err := c.open(ctx); err != nil {
		c.closed.Store(true)
		close(c.raw)
		return nil, err
	}
	return c, nil
}

// Close stops the camera. It's safe to call more than once.
func (c *Camera) Close() {
	if c.closed.Swap(true) {
		return
	}
	c.plat.close()
	unregister(c)
}

// Capture takes a picture at the camera's full resolution and returns
// it as an upright JPEG. The preview keeps running.
func (c *Camera) Capture(ctx context.Context) ([]byte, error) {
	if c.closed.Load() {
		return nil, ErrClosed
	}
	return c.plat.capture(ctx)
}

// HasTorch reports whether the camera has a light it can keep on.
func (c *Camera) HasTorch() bool {
	return !c.closed.Load() && c.plat.hasTorch()
}

// Torch turns the camera's light on or off.
func (c *Camera) Torch(on bool) error {
	if c.closed.Load() {
		return ErrClosed
	}
	if err := c.plat.setTorch(on); err != nil {
		return err
	}
	c.torch.Store(on)
	return nil
}

// TorchOn reports whether the light is on.
func (c *Camera) TorchOn() bool { return c.torch.Load() }

// Focus focuses (and meters exposure) at p, a point in the upright
// frame from (0, 0) at the top left to (1, 1) at the bottom right; the
// preview's tap-to-focus calls it. Cameras that can't focus on a point
// ignore it.
func (c *Camera) Focus(p f32.Point) error {
	if c.closed.Load() {
		return ErrClosed
	}
	p.X, p.Y = clamp01(p.X), clamp01(p.Y)
	return c.plat.focus(p)
}

// Size returns the preview frames' size, upright: zero until the first
// frame arrives.
func (c *Camera) Size() image.Point {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.size
}

// reportError passes a running camera's error to OnError, or logs it.
func (c *Camera) reportError(err error) {
	if c.opts.OnError != nil {
		c.opts.OnError(err)
		return
	}
	log.Printf("%v", err)
}

// deliver hands a frame from the platform to the converter, dropping
// it if the converter is still busy with the previous one.
func (c *Camera) deliver(f rawFrame) {
	if c.closed.Load() {
		return
	}
	select {
	case c.raw <- f:
	default:
	}
}

// convert turns platform frames into preview images and analysis
// frames.
func (c *Camera) convert() {
	for f := range c.raw {
		if c.closed.Load() {
			return
		}
		now := time.Now()
		c.mu.Lock()
		i := c.nextBuffer()
		c.mu.Unlock()
		var img *image.RGBA
		if f.bgra != nil {
			img = bgraToRGBA(c.bufs[i], f.bgra, f.w, f.h, f.stride)
		} else {
			img = yuvToRGBA(c.bufs[i], f)
		}
		c.mu.Lock()
		c.bufs[i] = img
		c.latest = i
		c.size = img.Rect.Size()
		c.frames++
		c.mu.Unlock()
		c.app.Invalidate()

		if c.opts.OnFrame == nil || c.opts.AnalysisSize < 0 {
			continue
		}
		if now.Sub(c.lastAn) < time.Second/time.Duration(c.opts.AnalysisRate) || c.analysis.Load() {
			continue
		}
		c.lastAn = now
		var gray *image.Gray
		if f.bgra != nil {
			gray = grayFromRGBA(img, c.opts.AnalysisSize)
		} else {
			gray = grayFromY(f, c.opts.AnalysisSize)
		}
		c.analysis.Store(true)
		go func() {
			defer c.analysis.Store(false)
			c.opts.OnFrame(Frame{Image: gray, Time: now})
		}()
	}
}

// nextBuffer picks a buffer that's neither drawn nor the latest.
func (c *Camera) nextBuffer() int {
	for i := range c.bufs {
		if i != c.latest && i != c.showing {
			return i
		}
	}
	return 0
}

// PreviewOptions configure Preview.
type PreviewOptions struct {
	// Fill crops the frame to fill the element, as camera apps do;
	// otherwise the whole frame shows, letterboxed.
	Fill bool
	// NoTapFocus turns off focusing where the user taps.
	NoTapFocus bool
	// Overlay draws over the preview: toView converts a point in the
	// frame, from (0, 0) to (1, 1), to the element's pixels, so a page
	// outline found in an analysis frame lands on the page.
	Overlay func(gtx layout.Context, toView func(f32.Point) f32.Point)
}

// Preview returns an element showing the live picture. It fills the
// space it's given.
func (c *Camera) Preview(opts PreviewOptions) giode.Element {
	return giode.Raw(func(gtx layout.Context) layout.Dimensions {
		return c.layoutPreview(gtx, opts)
	})
}

func (c *Camera) layoutPreview(gtx layout.Context, opts PreviewOptions) layout.Dimensions {
	view := gtx.Constraints.Max
	c.mu.Lock()
	var img *image.RGBA
	if c.latest >= 0 {
		c.showing = c.latest
		img = c.bufs[c.showing]
	}
	c.mu.Unlock()
	defer clip.Rect{Max: view}.Push(gtx.Ops).Pop()
	paint.Fill(gtx.Ops, blackColor)
	if img == nil {
		return layout.Dimensions{Size: view}
	}
	fit := fitFrame(img.Rect.Size(), view, opts.Fill)
	toView := func(p f32.Point) f32.Point {
		return f32.Pt(fit.Min.X+p.X*fit.Dx(), fit.Min.Y+p.Y*fit.Dy())
	}

	io := paint.NewImageOp(img)
	io.Filter = paint.FilterLinear
	sz := img.Rect.Size()
	tr := f32.Affine2D{}.Scale(f32.Point{}, f32.Pt(fit.Dx()/float32(sz.X), fit.Dy()/float32(sz.Y))).Offset(fit.Min)
	t := op.Affine(tr).Push(gtx.Ops)
	io.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	t.Pop()

	if !opts.NoTapFocus {
		for {
			ev, ok := gtx.Source.Event(pointer.Filter{Target: c, Kinds: pointer.Press | pointer.Release})
			if !ok {
				break
			}
			if e, ok := ev.(pointer.Event); ok && e.Kind == pointer.Release {
				p := f32.Pt((e.Position.X-fit.Min.X)/fit.Dx(), (e.Position.Y-fit.Min.Y)/fit.Dy())
				if p.X >= 0 && p.X <= 1 && p.Y >= 0 && p.Y <= 1 {
					go c.Focus(p)
				}
			}
		}
		pointer.CursorPointer.Add(gtx.Ops)
		eventArea(gtx, c)
	}
	if opts.Overlay != nil {
		opts.Overlay(gtx, toView)
	}
	return layout.Dimensions{Size: view}
}

// fitFrame places a frame of size frame in view: covering it (fill) or
// inside it, centred.
func fitFrame(frame, view image.Point, fill bool) frameRect {
	if frame.X == 0 || frame.Y == 0 {
		return frameRect{}
	}
	sx := float32(view.X) / float32(frame.X)
	sy := float32(view.Y) / float32(frame.Y)
	s := min(sx, sy)
	if fill {
		s = max(sx, sy)
	}
	w, h := float32(frame.X)*s, float32(frame.Y)*s
	return frameRect{Min: f32.Pt((float32(view.X)-w)/2, (float32(view.Y)-h)/2), W: w, H: h}
}

// frameRect is where the frame is drawn in the view.
type frameRect struct {
	Min  f32.Point
	W, H float32
}

func (r frameRect) Dx() float32 { return r.W }
func (r frameRect) Dy() float32 { return r.H }

func clamp01(v float32) float32 { return max(0, min(v, 1)) }

// requests pairs native callbacks with the cameras and requests they
// answer.
var (
	camMu   sync.Mutex
	cams    = map[int]*Camera{}
	nextCam int
)

func register(c *Camera) int {
	camMu.Lock()
	defer camMu.Unlock()
	nextCam++
	cams[nextCam] = c
	return nextCam
}

func unregister(c *Camera) {
	camMu.Lock()
	defer camMu.Unlock()
	for id, x := range cams {
		if x == c {
			delete(cams, id)
		}
	}
}

func lookup(id int) *Camera {
	camMu.Lock()
	defer camMu.Unlock()
	return cams[id]
}
