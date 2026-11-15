// Package raster rasterizes vector shapes - icons, glyphs - into
// images on the CPU, once, so they're drawn as cached textures instead
// of paths the GPU renderer rasterizes again every frame.
package raster

import (
	"image"
	"image/color"
	"math"
	"sort"
	"sync"

	"gioui.org/f32"
	"gioui.org/op"
	"gioui.org/op/paint"
	"github.com/srwiley/rasterx"
	"golang.org/x/image/math/fixed"

	"github.com/xmhrnx/giode/internal/linear"
)

// Segment is a path command: 'M', 'L', 'Q', 'C' or 'Z', with its points.
type Segment struct {
	Op  byte
	Pts [3]f32.Point
}

// Shape is a path filled and/or stroked. A zero-alpha color skips that
// part.
type Shape struct {
	Path        []Segment
	Fill        color.NRGBA
	Stroke      color.NRGBA
	StrokeWidth float32
	// Round gives strokes round caps and joins; otherwise butt caps and
	// miter joins.
	Round bool
}

// Render draws shapes, transformed by xf, into a new image of size.
func Render(size image.Point, xf f32.Affine2D, shapes []Shape) *image.RGBA {
	img := image.NewRGBA(image.Rectangle{Max: size})
	if size.X <= 0 || size.Y <= 0 {
		return img
	}
	scanner := rasterx.NewScannerGV(size.X, size.Y, img, img.Bounds())
	scanner.SetWinding(true)
	pt := func(p f32.Point) fixed.Point26_6 {
		q := xf.Transform(p)
		return fixed.Point26_6{X: fixed.Int26_6(q.X * 64), Y: fixed.Int26_6(q.Y * 64)}
	}
	// The stroke width scales with the transform's (uniform) scale.
	sx := xf.Transform(f32.Pt(1, 0)).Sub(xf.Transform(f32.Pt(0, 0)))
	scale := float32(math.Hypot(float64(sx.X), float64(sx.Y)))
	for _, s := range shapes {
		if s.Fill.A > 0 {
			f := rasterx.NewFiller(size.X, size.Y, scanner)
			f.SetColor(s.Fill)
			addPath(f, s.Path, pt)
			f.Draw()
			scanner.Clear()
		}
		if s.Stroke.A > 0 && s.StrokeWidth > 0 {
			st := rasterx.NewStroker(size.X, size.Y, scanner)
			w := fixed.Int26_6(s.StrokeWidth * scale * 64)
			if s.Round {
				st.SetStroke(w, 4*64, rasterx.RoundCap, nil, rasterx.RoundGap, rasterx.Round)
			} else {
				st.SetStroke(w, 4*64, rasterx.ButtCap, nil, rasterx.FlatGap, rasterx.Miter)
			}
			st.SetColor(s.Stroke)
			addPath(st, s.Path, pt)
			st.Draw()
			scanner.Clear()
		}
	}
	return img
}

func addPath(a rasterx.Adder, path []Segment, pt func(f32.Point) fixed.Point26_6) {
	open := false
	for _, s := range path {
		switch s.Op {
		case 'M':
			if open {
				a.Stop(false)
			}
			a.Start(pt(s.Pts[0]))
			open = true
		case 'L':
			a.Line(pt(s.Pts[0]))
		case 'Q':
			a.QuadBezier(pt(s.Pts[0]), pt(s.Pts[1]))
		case 'C':
			a.CubeBezier(pt(s.Pts[0]), pt(s.Pts[1]), pt(s.Pts[2]))
		case 'Z':
			if open {
				a.Stop(true)
				open = false
			}
		}
	}
	if open {
		a.Stop(false)
	}
}

type entry struct {
	op    paint.ImageOp
	bytes int
	used  uint64
}

var cache struct {
	sync.Mutex
	m     map[any]*entry
	bytes int
	tick  uint64
}

// maxBytes bounds the cached images; least recently used go first.
const maxBytes = 16 << 20

// Image returns the cached image for key, rendering it with render the
// first time. key must be comparable and include everything the image
// depends on (shape, size, colors).
func Image(key any, render func() *image.RGBA) paint.ImageOp {
	cache.Lock()
	cache.tick++
	if e, ok := cache.m[key]; ok {
		e.used = cache.tick
		cache.Unlock()
		return e.op
	}
	cache.Unlock()
	img := render()
	// Drawn premultiplied in sRGB; the GPU blends in linear light.
	linear.FixRGBA(img)
	op := paint.NewImageOp(img)
	op.Filter = paint.FilterNearest
	cache.Lock()
	defer cache.Unlock()
	if cache.m == nil {
		cache.m = make(map[any]*entry)
	}
	cache.m[key] = &entry{op: op, bytes: len(img.Pix), used: cache.tick}
	cache.bytes += len(img.Pix)
	if cache.bytes > maxBytes {
		evict()
	}
	return op
}

func evict() {
	type kv struct {
		k    any
		used uint64
	}
	all := make([]kv, 0, len(cache.m))
	for k, e := range cache.m {
		all = append(all, kv{k, e.used})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].used < all[j].used })
	for _, e := range all {
		if cache.bytes <= maxBytes*3/4 {
			break
		}
		cache.bytes -= cache.m[e.k].bytes
		delete(cache.m, e.k)
	}
}

// Draw paints a cached image at the origin.
func Draw(ops *op.Ops, img paint.ImageOp) {
	img.Add(ops)
	paint.PaintOp{}.Add(ops)
}
