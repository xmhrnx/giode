package camera

import (
	"image"
	"image/color"

	"gioui.org/io/event"
	"gioui.org/layout"
)

var blackColor = color.NRGBA{A: 0xff}

// eventArea makes the current clip area deliver pointer events to tag.
func eventArea(gtx layout.Context, tag event.Tag) {
	event.Op(gtx.Ops, tag)
}

// uprightSize is a frame's size once turned by rotation degrees.
func uprightSize(w, h, rotation int) (int, int) {
	if rotation == 90 || rotation == 270 {
		return h, w
	}
	return w, h
}

// sourceMap returns the source pixel for upright pixel (u, v) as
// sx = x0 + u*dxu + v*dxv, sy = y0 + u*dyu + v*dyv: the frame turned
// clockwise by rotation, then mirrored left to right if mirror.
type sourceMap struct{ x0, dxu, dxv, y0, dyu, dyv int }

func newSourceMap(w, h, rotation int, mirror bool) sourceMap {
	var m sourceMap
	switch rotation {
	case 90:
		// u = h-1-sy, v = sx
		m = sourceMap{x0: 0, dxu: 0, dxv: 1, y0: h - 1, dyu: -1, dyv: 0}
	case 180:
		m = sourceMap{x0: w - 1, dxu: -1, dxv: 0, y0: h - 1, dyu: 0, dyv: -1}
	case 270:
		// u = sy, v = w-1-sx
		m = sourceMap{x0: w - 1, dxu: 0, dxv: -1, y0: 0, dyu: 1, dyv: 0}
	default:
		m = sourceMap{x0: 0, dxu: 1, dxv: 0, y0: 0, dyu: 0, dyv: 1}
	}
	if mirror {
		uw, _ := uprightSize(w, h, rotation)
		// u' = uw-1-u
		m.x0 += (uw - 1) * m.dxu
		m.y0 += (uw - 1) * m.dyu
		m.dxu, m.dyu = -m.dxu, -m.dyu
	}
	return m
}

// reuse returns dst if it has size w x h, else a new image.
func reuse(dst *image.RGBA, w, h int) *image.RGBA {
	if dst != nil && dst.Rect.Dx() == w && dst.Rect.Dy() == h {
		return dst
	}
	return image.NewRGBA(image.Rect(0, 0, w, h))
}

// yuvToRGBA converts a YUV 4:2:0 frame (BT.601, full range, as Android
// cameras give) to an upright RGBA image, reusing dst when it fits.
func yuvToRGBA(dst *image.RGBA, f rawFrame) *image.RGBA {
	uw, uh := uprightSize(f.w, f.h, f.rotation)
	img := reuse(dst, uw, uh)
	m := newSourceMap(f.w, f.h, f.rotation, f.mirror)
	ps := f.uvPixStride
	if ps <= 0 {
		ps = 1
	}
	for v := 0; v < uh; v++ {
		sx := m.x0 + v*m.dxv
		sy := m.y0 + v*m.dyv
		row := img.Pix[v*img.Stride : v*img.Stride+uw*4]
		for u := 0; u < uw; u++ {
			yy := int(f.y[sy*f.yStride+sx])
			ci := (sy>>1)*f.uvStride + (sx>>1)*ps
			cb := int(f.u[ci]) - 128
			cr := int(f.v[ci]) - 128
			// Fixed point, 16 fractional bits.
			r := yy + (91881*cr)>>16
			g := yy - (22554*cb+46802*cr)>>16
			b := yy + (116130*cb)>>16
			o := u * 4
			row[o] = clamp8(r)
			row[o+1] = clamp8(g)
			row[o+2] = clamp8(b)
			row[o+3] = 0xff
			sx += m.dxu
			sy += m.dyu
		}
	}
	return img
}

// bgraToRGBA copies an upright BGRA frame (as Apple cameras give) into
// an RGBA image.
func bgraToRGBA(dst *image.RGBA, bgra []byte, w, h, stride int) *image.RGBA {
	img := reuse(dst, w, h)
	for y := 0; y < h; y++ {
		src := bgra[y*stride : y*stride+w*4]
		row := img.Pix[y*img.Stride : y*img.Stride+w*4]
		for x := 0; x < w*4; x += 4 {
			row[x] = src[x+2]
			row[x+1] = src[x+1]
			row[x+2] = src[x]
			row[x+3] = 0xff
		}
	}
	return img
}

// analysisSize scales w x h so its long side is long, never up.
func analysisSize(w, h, long int) (int, int) {
	if long <= 0 || (w <= long && h <= long) {
		return w, h
	}
	if w >= h {
		return long, max(1, h*long/w)
	}
	return max(1, w*long/h), long
}

// grayFromY samples a frame's brightness plane into an upright grey
// image with long pixels on its long side.
func grayFromY(f rawFrame, long int) *image.Gray {
	uw, uh := uprightSize(f.w, f.h, f.rotation)
	gw, gh := analysisSize(uw, uh, long)
	g := image.NewGray(image.Rect(0, 0, gw, gh))
	m := newSourceMap(f.w, f.h, f.rotation, f.mirror)
	for y := 0; y < gh; y++ {
		v := y * uh / gh
		for x := 0; x < gw; x++ {
			u := x * uw / gw
			sx := m.x0 + u*m.dxu + v*m.dxv
			sy := m.y0 + u*m.dyu + v*m.dyv
			g.Pix[y*g.Stride+x] = f.y[sy*f.yStride+sx]
		}
	}
	return g
}

// grayFromRGBA samples an upright RGBA preview into a grey image.
func grayFromRGBA(img *image.RGBA, long int) *image.Gray {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	gw, gh := analysisSize(w, h, long)
	g := image.NewGray(image.Rect(0, 0, gw, gh))
	for y := 0; y < gh; y++ {
		sy := y * h / gh
		for x := 0; x < gw; x++ {
			sx := x * w / gw
			p := img.Pix[sy*img.Stride+sx*4:]
			// BT.601 luma.
			g.Pix[y*g.Stride+x] = uint8((19595*int(p[0]) + 38470*int(p[1]) + 7471*int(p[2]) + 1<<15) >> 16)
		}
	}
	return g
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
