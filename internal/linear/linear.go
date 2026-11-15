// Package linear prepares images with transparency for Gio's GPU
// renderer, which decodes textures from sRGB and blends in linear
// light: their pixels must be premultiplied in linear light, then
// encoded. Go's images are premultiplied in sRGB (image.RGBA) or not at
// all (image.NRGBA, which Gio premultiplies in sRGB as it uploads it);
// either way every partly transparent pixel comes out darker than it
// should - a dark fringe around light text, icons and shapes.
package linear

import (
	"image"
	"image/draw"
	"math"
)

var (
	decode [256]float32 // sRGB byte -> linear
	encode [4097]uint8  // linear in 4096ths -> sRGB byte
)

func init() {
	for i := range decode {
		v := float64(i) / 255
		if v <= 0.04045 {
			v /= 12.92
		} else {
			v = math.Pow((v+0.055)/1.055, 2.4)
		}
		decode[i] = float32(v)
	}
	for i := range encode {
		v := float64(i) / 4096
		if v <= 0.0031308 {
			v *= 12.92
		} else {
			v = 1.055*math.Pow(v, 1/2.4) - 0.055
		}
		encode[i] = uint8(math.Round(v * 255))
	}
}

// Premul returns the sRGB byte of channel c (straight, sRGB) at alpha a
// premultiplied in linear light.
func Premul(c, a uint8) uint8 {
	return encode[int(decode[c]*float32(a)/255*4096+0.5)]
}

// FixRGBA converts img, premultiplied in sRGB as Go's image.RGBA is,
// to premultiplied in linear light, in place.
func FixRGBA(img *image.RGBA) {
	w := img.Rect.Dx() * 4
	for y := 0; y < img.Rect.Dy(); y++ {
		row := img.Pix[y*img.Stride : y*img.Stride+w]
		for i := 0; i < len(row); i += 4 {
			a := row[i+3]
			if a == 0 || a == 0xff {
				continue
			}
			for c := 0; c < 3; c++ {
				// Unpremultiply in sRGB, then premultiply in linear.
				s := min(int(row[i+c])*255/int(a), 255)
				row[i+c] = Premul(uint8(s), a)
			}
		}
	}
}

// Image returns img ready to upload: img itself when it's opaque,
// otherwise a copy premultiplied in linear light.
func Image(img image.Image) image.Image {
	if o, ok := img.(interface{ Opaque() bool }); ok && o.Opaque() {
		return img
	}
	b := img.Bounds()
	src, ok := img.(*image.NRGBA)
	if !ok {
		src = image.NewNRGBA(b)
		draw.Draw(src, b, img, b.Min, draw.Src)
	}
	out := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	w := b.Dx() * 4
	for y := 0; y < b.Dy(); y++ {
		in := src.Pix[(y+b.Min.Y-src.Rect.Min.Y)*src.Stride+(b.Min.X-src.Rect.Min.X)*4:][:w]
		o := out.Pix[y*out.Stride:][:w]
		for i := 0; i < w; i += 4 {
			a := in[i+3]
			switch a {
			case 0:
			case 0xff:
				copy(o[i:i+4], in[i:i+4])
			default:
				o[i], o[i+1], o[i+2], o[i+3] = Premul(in[i], a), Premul(in[i+1], a), Premul(in[i+2], a), a
			}
		}
	}
	return out
}
