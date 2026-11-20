// Package video shows moving pictures: a Surface takes frames from any
// goroutine (a decoder, a call's remote track, a screen capture) and
// its View element draws the latest one.
//
//	s := video.NewSurface(app)
//	go func() {
//		for f := range decoded {
//			s.PushI420(f) // or s.Push(rgba)
//		}
//	}()
//	...
//	s.View(video.ViewOptions{Fit: video.Cover, Mirror: true})
//
// Frames are double buffered: pushing never waits for drawing, and a
// frame being drawn is never written. The GPU texture is uploaded once
// per new frame, not once per redraw.
package video

import (
	"image"
	"image/color"
	"sync"
	"time"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/xmhrnx/giode"
)

// Fit is how a frame fills the view.
type Fit int

const (
	// Contain shows the whole frame, letterboxed.
	Contain Fit = iota
	// Cover fills the view, cropping the frame.
	Cover
	// Stretch fills the view, distorting the frame.
	Stretch
)

// Surface holds the latest frame of a video.
type Surface struct {
	app *giode.App

	mu sync.Mutex
	// bufs are the frame buffers: latest is the newest frame, showing
	// the one the view last drew (the GPU may still read it).
	bufs            [3]*image.RGBA
	latest, showing int
	gen             uint64 // bumped with each frame
	// The view's cached image op, for gen.
	op    paint.ImageOp
	opGen uint64
	// Frame rate over the last second.
	frames  int
	times   [32]time.Time
	timesAt int
}

// NewSurface returns an empty surface. Pushing a frame redraws app's
// window; app may be nil when something else redraws it.
func NewSurface(app *giode.App) *Surface {
	return &Surface{app: app, latest: -1, showing: -1}
}

// Push shows img. It's copied, so the caller may reuse it.
func (s *Surface) Push(img image.Image) {
	b := img.Bounds()
	dst := s.buffer(b.Dx(), b.Dy())
	if src, ok := img.(*image.RGBA); ok {
		for y := 0; y < b.Dy(); y++ {
			o := src.PixOffset(b.Min.X, b.Min.Y+y)
			copy(dst.Pix[y*dst.Stride:y*dst.Stride+b.Dx()*4], src.Pix[o:])
		}
	} else {
		for y := 0; y < b.Dy(); y++ {
			for x := 0; x < b.Dx(); x++ {
				c := color.RGBAModel.Convert(img.At(b.Min.X+x, b.Min.Y+y)).(color.RGBA)
				i := y*dst.Stride + x*4
				dst.Pix[i], dst.Pix[i+1], dst.Pix[i+2], dst.Pix[i+3] = c.R, c.G, c.B, c.A
			}
		}
	}
	s.publish(dst)
}

// PushBGRA shows w×h BGRA pixels, rows stride bytes apart: what Apple
// and Windows frameworks produce.
func (s *Surface) PushBGRA(pix []byte, w, h, stride int) {
	dst := s.buffer(w, h)
	for y := 0; y < h; y++ {
		src := pix[y*stride : y*stride+w*4]
		out := dst.Pix[y*dst.Stride:]
		for x := 0; x < len(src); x += 4 {
			out[x], out[x+1], out[x+2], out[x+3] = src[x+2], src[x+1], src[x], 255
		}
	}
	s.publish(dst)
}

// I420 is a planar YUV 4:2:0 frame, what video decoders and WebRTC
// produce.
type I420 struct {
	Y, U, V                   []byte
	StrideY, StrideU, StrideV int
	// PixelStrideUV is the distance between chroma samples: 1 for
	// planar I420, 2 for the interleaved planes of NV12 and NV21 (as
	// Android's YUV_420_888 images often are). 0 means 1.
	PixelStrideUV int
	Width, Height int
	// FullRange is set for full-range (JPEG) levels; video is limited
	// range (16–235) by default.
	FullRange bool
	// BT709 selects the HD matrix; BT.601 by default.
	BT709 bool
}

// PushI420 converts f and shows it.
func (s *Surface) PushI420(f I420) {
	dst := s.buffer(f.Width, f.Height)
	convertI420(dst, f)
	s.publish(dst)
}

// Clear removes the frame: the view shows its placeholder.
func (s *Surface) Clear() {
	s.mu.Lock()
	s.latest = -1
	s.gen++
	s.mu.Unlock()
	s.invalidate()
}

// Size is the latest frame's size; zero before the first.
func (s *Surface) Size() image.Point {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.latest < 0 {
		return image.Point{}
	}
	return s.bufs[s.latest].Rect.Size()
}

// Snapshot returns a copy of the latest frame, or nil.
func (s *Surface) Snapshot() *image.RGBA {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.latest < 0 {
		return nil
	}
	src := s.bufs[s.latest]
	img := image.NewRGBA(src.Rect)
	copy(img.Pix, src.Pix)
	return img
}

// Frames is how many frames were pushed.
func (s *Surface) Frames() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.frames
}

// FPS is the rate frames arrived at over the last second.
func (s *Surface) FPS() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	n := 0
	var oldest time.Time
	for _, t := range s.times {
		if !t.IsZero() && now.Sub(t) <= time.Second {
			n++
			if oldest.IsZero() || t.Before(oldest) {
				oldest = t
			}
		}
	}
	if n < 2 {
		return float64(n)
	}
	return float64(n-1) / now.Sub(oldest).Seconds()
}

// buffer returns a w×h buffer that's neither the latest frame nor the
// one being drawn.
func (s *Surface) buffer(w, h int) *image.RGBA {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.bufs {
		if i == s.latest || i == s.showing {
			continue
		}
		b := s.bufs[i]
		if b == nil || b.Rect.Dx() != w || b.Rect.Dy() != h {
			b = image.NewRGBA(image.Rect(0, 0, w, h))
			s.bufs[i] = b
		}
		return b
	}
	panic("unreachable")
}

func (s *Surface) publish(img *image.RGBA) {
	s.mu.Lock()
	for i, b := range s.bufs {
		if b == img {
			s.latest = i
		}
	}
	s.gen++
	s.frames++
	s.times[s.timesAt] = time.Now()
	s.timesAt = (s.timesAt + 1) % len(s.times)
	s.mu.Unlock()
	s.invalidate()
}

func (s *Surface) invalidate() {
	if s.app != nil {
		s.app.Invalidate()
	}
}

// ViewOptions configure View.
type ViewOptions struct {
	Fit Fit
	// Mirror flips the frame left to right, as a self view is.
	Mirror bool
	// Rotation turns the frame clockwise: 0, 90, 180 or 270 degrees.
	Rotation int
	// Background fills the view around and under the frame; black by
	// default.
	Background color.NRGBA
	// Placeholder shows before the first frame and after Clear.
	Placeholder giode.Element
	// Label is read by screen readers.
	Label string
	// Overlay draws over the frame: toView converts a point in the
	// frame, from (0, 0) to (1, 1), to the view's pixels.
	Overlay func(gtx layout.Context, toView func(f32.Point) f32.Point)
}

// View returns an element showing the latest frame. It fills the space
// it's given.
func (s *Surface) View(opts ViewOptions) giode.Element {
	if opts.Background == (color.NRGBA{}) {
		opts.Background = color.NRGBA{A: 255}
	}
	e := giode.Raw(func(gtx layout.Context) layout.Dimensions { return s.layout(gtx, opts) })
	if opts.Label != "" {
		e = giode.AccessibleGroup(giode.SemanticNode{Kind: giode.SemanticImage, Label: opts.Label}, e)
	}
	return e
}

func (s *Surface) layout(gtx layout.Context, opts ViewOptions) layout.Dimensions {
	view := gtx.Constraints.Max
	s.mu.Lock()
	var (
		img *image.RGBA
		io  paint.ImageOp
	)
	if s.latest >= 0 {
		s.showing = s.latest
		img = s.bufs[s.showing]
		if s.opGen != s.gen {
			// A new frame: a new op, so the GPU uploads it once.
			s.op = paint.NewImageOp(img)
			s.op.Filter = paint.FilterLinear
			s.opGen = s.gen
		}
		io = s.op
	}
	s.mu.Unlock()

	defer clip.Rect{Max: view}.Push(gtx.Ops).Pop()
	paint.Fill(gtx.Ops, opts.Background)
	if img == nil {
		if opts.Placeholder != nil {
			gtx.Constraints.Min = view
			opts.Placeholder.Layout(gtx)
		}
		return layout.Dimensions{Size: view}
	}
	sz := img.Rect.Size()
	tr, r := placement(sz, view, opts)
	t := op.Affine(tr).Push(gtx.Ops)
	io.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	t.Pop()
	if opts.Overlay != nil {
		opts.Overlay(gtx, func(p f32.Point) f32.Point {
			if opts.Mirror {
				p.X = 1 - p.X
			}
			return f32.Pt(r.Min.X+p.X*r.Dx(), r.Min.Y+p.Y*r.Dy())
		})
	}
	return layout.Dimensions{Size: view}
}

// placement returns the transform drawing a frame of size sz in view,
// and where the upright frame lands.
func placement(sz, view image.Point, opts ViewOptions) (f32.Affine2D, frect) {
	rot := ((opts.Rotation % 360) + 360) % 360 / 90 * 90
	up := sz
	if rot == 90 || rot == 270 {
		up = image.Pt(sz.Y, sz.X)
	}
	r := fit(up, view, opts.Fit)
	if up.X == 0 || up.Y == 0 {
		return f32.Affine2D{}, r
	}
	// Frame pixels → centred at the origin, turned, mirrored, scaled
	// to the upright frame's place.
	c := f32.Pt(float32(sz.X)/2, float32(sz.Y)/2)
	t := f32.Affine2D{}.Offset(c.Mul(-1))
	if rot != 0 {
		t = t.Rotate(f32.Point{}, float32(rot)*3.14159265/180)
	}
	sx, sy := r.Dx()/float32(up.X), r.Dy()/float32(up.Y)
	if opts.Mirror {
		sx = -sx
	}
	t = t.Scale(f32.Point{}, f32.Pt(sx, sy))
	t = t.Offset(f32.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2))
	return t, r
}

// fit places a frame of size frame in view.
func fit(frame, view image.Point, mode Fit) frect {
	if frame.X == 0 || frame.Y == 0 {
		return frect{}
	}
	vw, vh := float32(view.X), float32(view.Y)
	if mode == Stretch {
		return frect{Max: f32.Pt(vw, vh)}
	}
	sx, sy := vw/float32(frame.X), vh/float32(frame.Y)
	s := min(sx, sy)
	if mode == Cover {
		s = max(sx, sy)
	}
	w, h := float32(frame.X)*s, float32(frame.Y)*s
	min := f32.Pt((vw-w)/2, (vh-h)/2)
	return frect{Min: min, Max: min.Add(f32.Pt(w, h))}
}

// frect is where a frame is drawn in a view, in pixels.
type frect struct{ Min, Max f32.Point }

func (r frect) Dx() float32 { return r.Max.X - r.Min.X }
func (r frect) Dy() float32 { return r.Max.Y - r.Min.Y }
