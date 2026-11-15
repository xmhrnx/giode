package properties

import (
	"image"
	"image/color"
	"math"
	"sync"

	"gioui.org/f32"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/xmhrnx/giode/internal/linear"
)

// BoxShadow is a drop shadow painted behind an element (CSS
// box-shadow). A zero alpha color means no shadow. Lengths are in dp.
type BoxShadow struct {
	// X and Y offset the shadow from the element.
	X, Y int
	// Blur is the shadow's soft-edge radius. Zero gives a crisp edge.
	Blur int
	// Spread grows (or, when negative, shrinks) the shadow before it
	// is blurred.
	Spread int
	// Color is the shadow color; its alpha scales the shadow.
	Color string
}

// PaintBoxShadow paints the shadow for an element whose painted box
// is rect with the given corner radius.
func PaintBoxShadow(ops *op.Ops, rect image.Rectangle, radius BorderRadius, s BoxShadow) {
	PaintBoxShadowCorners(ops, rect, UniformCorners(int(radius)), s)
}

// PaintBoxShadowCorners paints the shadow for an element whose painted
// box is rect with per-corner radii. A blurred shadow is a Gaussian
// blur of the rounded rectangle (CSS's, with a standard deviation of
// half the blur radius), blurred once on the CPU for its corner radii
// and blur and cached as an image. It's drawn as nine slices - corners
// as they are, edges and middle stretched - so it costs the same for
// any box size: a few textured rectangles, not a path per layer.
func PaintBoxShadowCorners(ops *op.Ops, rect image.Rectangle, corners Corners, s BoxShadow) {
	paintBoxShadow(ops, rect, corners, s, false)
}

// PaintBoxShadowExact is PaintBoxShadowCorners for a box drawn under a
// scale or rotation: a crisp shadow is one path (see FillRRect).
func PaintBoxShadowExact(ops *op.Ops, rect image.Rectangle, corners Corners, s BoxShadow) {
	paintBoxShadow(ops, rect, corners, s, true)
}

func paintBoxShadow(ops *op.Ops, rect image.Rectangle, corners Corners, s BoxShadow, exact bool) {
	col := CalcColor(s.Color)
	if col.A == 0 || rect.Dx() <= 0 || rect.Dy() <= 0 {
		return
	}
	r := rect.Add(image.Pt(s.X, s.Y)).Inset(-s.Spread)
	if r.Dx() <= 0 || r.Dy() <= 0 {
		return
	}
	grow := func(v int) int {
		if v == 0 {
			return 0
		}
		return max(0, v+s.Spread)
	}
	c := clampCorners(Corners{grow(corners.TopLeft), grow(corners.TopRight), grow(corners.BottomRight), grow(corners.BottomLeft)}, r.Dx(), r.Dy())
	if s.Blur <= 0 {
		FillRRect(ops, r, c, exact, func() {
			paint.ColorOp{Color: col}.Add(ops)
			paint.PaintOp{}.Add(ops)
		})
		return
	}
	paintBlurred(ops, r, c, s.Blur, col)
}

// shadowKey identifies a blurred shadow image.
type shadowKey struct {
	c    Corners
	blur int
	col  color.NRGBA
	// size is set when the box is too small for nine slices and the
	// shadow is blurred at its full size.
	size image.Point
}

// shadowImage is a blurred rounded rectangle, cut into nine slices.
type shadowImage struct {
	// slices are the images of the corners, edges and middle, in rows
	// top to bottom, left to right; nil for an empty slice.
	slices [9]*paint.ImageOp
	// cols and rows are the source widths of the left, middle and right
	// columns, and heights of the top, middle and bottom rows.
	cols, rows [3]int
	used       uint64
}

var shadowCache struct {
	sync.Mutex
	m    map[shadowKey]*shadowImage
	tick uint64
}

// maxShadowImages bounds the cached shadow images.
const maxShadowImages = 64

// paintBlurred draws the Gaussian-blurred rounded rectangle r.
func paintBlurred(ops *op.Ops, r image.Rectangle, c Corners, blur int, col color.NRGBA) {
	// The blur reaches 3 standard deviations, 1.5 times the radius.
	m := int(math.Ceil(1.5 * float64(blur)))
	rl, rr := max(c.TopLeft, c.BottomLeft), max(c.TopRight, c.BottomRight)
	rt, rb := max(c.TopLeft, c.TopRight), max(c.BottomLeft, c.BottomRight)
	// The prototype has corner slices reaching m past the rounding on
	// each side, and a one pixel middle beyond the blur's reach.
	key := shadowKey{c: c, blur: blur, col: col}
	protoW, protoH := rl+rr+2*m+1, rt+rb+2*m+1
	if r.Dx() < protoW || r.Dy() < protoH {
		key.size = r.Size()
		protoW, protoH = r.Dx(), r.Dy()
	}
	img := cachedShadow(key, protoW, protoH, m, rl, rr, rt, rb)

	// Destination columns and rows, around r grown by the blur.
	dst := r.Inset(-m)
	dcols := [3]int{img.cols[0], dst.Dx() - img.cols[0] - img.cols[2], img.cols[2]}
	drows := [3]int{img.rows[0], dst.Dy() - img.rows[0] - img.rows[2], img.rows[2]}
	y := dst.Min.Y
	for row := 0; row < 3; row++ {
		x := dst.Min.X
		for col := 0; col < 3; col++ {
			w, h := dcols[col], drows[row]
			if s := img.slices[row*3+col]; s != nil && w > 0 && h > 0 {
				sw, sh := img.cols[col], img.rows[row]
				st := clip.Rect{Min: image.Pt(x, y), Max: image.Pt(x+w, y+h)}.Push(ops)
				tr := f32.Affine2D{}.Scale(f32.Point{}, f32.Pt(float32(w)/float32(sw), float32(h)/float32(sh))).
					Offset(f32.Pt(float32(x), float32(y)))
				t := op.Affine(tr).Push(ops)
				s.Add(ops)
				paint.PaintOp{}.Add(ops)
				t.Pop()
				st.Pop()
			}
			x += w
		}
		y += drows[row]
	}
}

// cachedShadow returns the sliced shadow image for key, blurring it on
// first use.
func cachedShadow(key shadowKey, w, h, m, rl, rr, rt, rb int) *shadowImage {
	shadowCache.Lock()
	defer shadowCache.Unlock()
	shadowCache.tick++
	if img, ok := shadowCache.m[key]; ok {
		img.used = shadowCache.tick
		return img
	}
	if shadowCache.m == nil {
		shadowCache.m = make(map[shadowKey]*shadowImage)
	}
	if len(shadowCache.m) >= maxShadowImages {
		// Drop the least recently used.
		var oldest shadowKey
		var at uint64 = math.MaxUint64
		for k, v := range shadowCache.m {
			if v.used < at {
				oldest, at = k, v.used
			}
		}
		delete(shadowCache.m, oldest)
	}
	full := blurredRRect(w, h, m, key.c, key.blur, key.col)
	img := &shadowImage{used: shadowCache.tick}
	if key.size != (image.Point{}) {
		// One slice: the whole shadow.
		img.cols = [3]int{0, full.Bounds().Dx(), 0}
		img.rows = [3]int{0, full.Bounds().Dy(), 0}
		op := paint.NewImageOp(linear.Image(full))
		img.slices[4] = &op
	} else {
		img.cols = [3]int{2*m + rl, 1, 2*m + rr}
		img.rows = [3]int{2*m + rt, 1, 2*m + rb}
		y := 0
		for row := 0; row < 3; row++ {
			x := 0
			for col := 0; col < 3; col++ {
				sub := full.SubImage(image.Rect(x, y, x+img.cols[col], y+img.rows[row])).(*image.NRGBA)
				// A copy, so each slice is its own texture and edges
				// don't sample their neighbours.
				cp := image.NewNRGBA(image.Rect(0, 0, sub.Bounds().Dx(), sub.Bounds().Dy()))
				for yy := 0; yy < cp.Rect.Dy(); yy++ {
					copy(cp.Pix[yy*cp.Stride:yy*cp.Stride+cp.Rect.Dx()*4], sub.Pix[yy*sub.Stride:yy*sub.Stride+cp.Rect.Dx()*4])
				}
				op := paint.NewImageOp(linear.Image(cp))
				op.Filter = paint.FilterNearest
				img.slices[row*3+col] = &op
				x += img.cols[col]
			}
			y += img.rows[row]
		}
	}
	shadowCache.m[key] = img
	return img
}

// blurredRRect renders a w by h rounded rectangle with corners c,
// inside a margin of m, blurred with a Gaussian of standard deviation
// blur/2, in col.
func blurredRRect(w, h, m int, c Corners, blur int, col color.NRGBA) *image.NRGBA {
	W, H := w+2*m, h+2*m
	mask := make([]float32, W*H)
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			mask[y*W+x] = rrectCoverage(float32(x-m)+0.5, float32(y-m)+0.5, float32(w), float32(h), c)
		}
	}
	sigma := float64(blur) / 2
	gaussBlur(mask, W, H, sigma)
	out := image.NewNRGBA(image.Rect(0, 0, W, H))
	for i, a := range mask {
		out.Pix[i*4+0] = col.R
		out.Pix[i*4+1] = col.G
		out.Pix[i*4+2] = col.B
		out.Pix[i*4+3] = uint8(min(1, max(0, a))*float32(col.A) + 0.5)
	}
	return out
}

// rrectCoverage is how much of the pixel centered at (x, y) the w by h
// rounded rectangle at the origin covers, from its signed distance.
func rrectCoverage(x, y, w, h float32, c Corners) float32 {
	var rad float32
	var cx, cy float32
	switch {
	case x < float32(c.TopLeft) && y < float32(c.TopLeft):
		rad, cx, cy = float32(c.TopLeft), float32(c.TopLeft), float32(c.TopLeft)
	case x > w-float32(c.TopRight) && y < float32(c.TopRight):
		rad, cx, cy = float32(c.TopRight), w-float32(c.TopRight), float32(c.TopRight)
	case x > w-float32(c.BottomRight) && y > h-float32(c.BottomRight):
		rad, cx, cy = float32(c.BottomRight), w-float32(c.BottomRight), h-float32(c.BottomRight)
	case x < float32(c.BottomLeft) && y > h-float32(c.BottomLeft):
		rad, cx, cy = float32(c.BottomLeft), float32(c.BottomLeft), h-float32(c.BottomLeft)
	}
	var d float32
	if rad > 0 {
		dx, dy := x-cx, y-cy
		d = float32(math.Sqrt(float64(dx*dx+dy*dy))) - rad
	} else {
		d = max(-x, x-w, -y, y-h)
	}
	return min(1, max(0, 0.5-d))
}

// gaussBlur blurs a w by h mask in place, horizontally then vertically.
func gaussBlur(mask []float32, w, h int, sigma float64) {
	if sigma <= 0 {
		return
	}
	radius := int(math.Ceil(3 * sigma))
	kernel := make([]float32, 2*radius+1)
	var sum float32
	for i := range kernel {
		d := float64(i - radius)
		kernel[i] = float32(math.Exp(-d * d / (2 * sigma * sigma)))
		sum += kernel[i]
	}
	for i := range kernel {
		kernel[i] /= sum
	}
	tmp := make([]float32, len(mask))
	for y := 0; y < h; y++ {
		row := mask[y*w : y*w+w]
		for x := 0; x < w; x++ {
			var acc float32
			for k, kv := range kernel {
				if xx := x + k - radius; xx >= 0 && xx < w {
					acc += row[xx] * kv
				}
			}
			tmp[y*w+x] = acc
		}
	}
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			var acc float32
			for k, kv := range kernel {
				if yy := y + k - radius; yy >= 0 && yy < h {
					acc += tmp[yy*w+x] * kv
				}
			}
			mask[y*w+x] = acc
		}
	}
}
