package textrender

import (
	"image/color"
	"math"
	"sync"
)

// Glyph coverage is corrected for the GPU's blending, which happens in
// linear light: there, coverage makes dark text on a light background
// thin and gray and light text on a dark one heavy and soft, unlike
// the platforms' text, blended in the display's gamma. The correction
// depends on the text's color and the background's: coverage a is
// drawn with the alpha that, blended in linear light, gives the
// luminance blending in gamma gives. Dark text over a light background
// gets more coverage, light text over a dark one less - by its
// contrast with the background, not its color alone, so gray text on
// white is fattened like dark text and white text on a saturated
// button isn't thinned as if over black.

// Backgrounds text is drawn over: boxes push theirs (Box with a
// Background) while laying out their content.
var (
	bgStack []color.NRGBA
	// DefaultBackground is the background without a box's: the
	// theme's.
	DefaultBackground = func() color.NRGBA { return color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff} }
)

// PushBackground makes c the background of the text drawn until the
// returned function runs. A translucent c is composited over the
// current background; a transparent one changes nothing.
func PushBackground(c color.NRGBA) (pop func()) {
	if c.A == 0 {
		return func() {}
	}
	if c.A < 0xff {
		c = over(c, Background())
	}
	bgStack = append(bgStack, c)
	n := len(bgStack)
	return func() { bgStack = bgStack[:n-1] }
}

// Background returns the background text is drawn over now, opaque.
func Background() color.NRGBA {
	if n := len(bgStack); n > 0 {
		return bgStack[n-1]
	}
	c := DefaultBackground()
	c.A = 0xff
	return c
}

// over composites c over the opaque bg, in gamma as UI colors are
// specified.
func over(c, bg color.NRGBA) color.NRGBA {
	a := float64(c.A) / 255
	mix := func(f, b uint8) uint8 { return uint8(math.Round(float64(f)*a + float64(b)*(1-a))) }
	return color.NRGBA{R: mix(c.R, bg.R), G: mix(c.G, bg.G), B: mix(c.B, bg.B), A: 0xff}
}

func toLinear(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

// linLum is the relative luminance of the gamma-encoded r, g, b in
// [0, 1].
func linLum(r, g, b float64) float64 {
	return 0.2126*toLinear(r) + 0.7152*toLinear(g) + 0.0722*toLinear(b)
}

var luts sync.Map // [2]color.NRGBA (opaque) -> *[256]uint8

// coverageLUT maps a glyph's coverage to the alpha it's drawn with in
// col over bg.
func coverageLUT(col, bg color.NRGBA) *[256]uint8 {
	col.A, bg.A = 0xff, 0xff
	key := [2]color.NRGBA{col, bg}
	if l, ok := luts.Load(key); ok {
		return l.(*[256]uint8)
	}
	f := [3]float64{float64(col.R) / 255, float64(col.G) / 255, float64(col.B) / 255}
	b := [3]float64{float64(bg.R) / 255, float64(bg.G) / 255, float64(bg.B) / 255}
	lf, lb := linLum(f[0], f[1], f[2]), linLum(b[0], b[1], b[2])
	var l [256]uint8
	for i := range l {
		a := float64(i) / 255
		out := a
		if math.Abs(lf-lb) > 0.005 {
			// The luminance blending in gamma gives, reached in linear
			// light.
			t := linLum(f[0]*a+b[0]*(1-a), f[1]*a+b[1]*(1-a), f[2]*a+b[2]*(1-a))
			out = min(max((t-lb)/(lf-lb), 0), 1)
		}
		l[i] = uint8(math.Round(out * 255))
	}
	luts.Store(key, &l)
	return &l
}
