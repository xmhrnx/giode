package colorpicker

import (
	"image/color"
	"math"
)

// hsv is a color as hue (0-360), saturation, value and alpha (0-1).
type hsv struct{ h, s, v, a float64 }

// rgb returns c as 8-bit sRGB.
func (c hsv) rgb() color.NRGBA {
	h := math.Mod(c.h, 360)
	if h < 0 {
		h += 360
	}
	ch := c.v * c.s
	x := ch * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := c.v - ch
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = ch, x, 0
	case h < 120:
		r, g, b = x, ch, 0
	case h < 180:
		r, g, b = 0, ch, x
	case h < 240:
		r, g, b = 0, x, ch
	case h < 300:
		r, g, b = x, 0, ch
	default:
		r, g, b = ch, 0, x
	}
	to8 := func(f float64) uint8 { return uint8(math.Round(math.Min(math.Max(f, 0), 1) * 255)) }
	return color.NRGBA{R: to8(r + m), G: to8(g + m), B: to8(b + m), A: to8(c.a)}
}

// fromRGB returns col as HSV, keeping hue when col has none (a grey).
func fromRGB(col color.NRGBA, hue float64) hsv {
	r, g, b := float64(col.R)/255, float64(col.G)/255, float64(col.B)/255
	mx, mn := math.Max(r, math.Max(g, b)), math.Min(r, math.Min(g, b))
	d := mx - mn
	c := hsv{h: hue, v: mx, a: float64(col.A) / 255}
	if mx > 0 {
		c.s = d / mx
	}
	if d > 0 {
		switch mx {
		case r:
			c.h = 60 * math.Mod((g-b)/d, 6)
		case g:
			c.h = 60 * ((b-r)/d + 2)
		default:
			c.h = 60 * ((r-g)/d + 4)
		}
		if c.h < 0 {
			c.h += 360
		}
	}
	return c
}
