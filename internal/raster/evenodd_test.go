package raster

import (
	"image"
	"image/color"
	"math"
	"testing"

	"gioui.org/f32"
)

// circle is a closed polygonal circle; clockwise reverses its direction.
func circle(c f32.Point, r float32, clockwise bool) []Segment {
	const n = 32
	var out []Segment
	for i := 0; i <= n; i++ {
		a := float64(i) / n * 2 * 3.14159265
		if clockwise {
			a = -a
		}
		p := c.Add(f32.Pt(r*float32(cos(a)), r*float32(sin(a))))
		op := byte('L')
		if i == 0 {
			op = 'M'
		}
		out = append(out, Segment{Op: op, Pts: [3]f32.Point{p}})
	}
	return append(out, Segment{Op: 'Z'})
}

func TestEvenOddNested(t *testing.T) {
	c := f32.Pt(50, 50)
	// A target: three circles drawn the same way round.
	path := append(append(circle(c, 45, false), circle(c, 30, false)...), circle(c, 15, false)...)
	render := func(evenOdd bool) *image.RGBA {
		return Render(image.Pt(100, 100), f32.AffineId(), []Shape{{Path: path, Fill: color.NRGBA{A: 255}, EvenOdd: evenOdd}})
	}
	at := func(img *image.RGBA, x int) bool { return img.RGBAAt(x, 50).A > 128 }
	eo := render(true)
	// Outer band painted, middle band a hole, the middle disc painted.
	if !at(eo, 10) || at(eo, 28) || !at(eo, 50) {
		t.Errorf("evenodd: outer %v, middle %v, center %v; want painted, hole, painted", at(eo, 10), at(eo, 28), at(eo, 50))
	}
	nz := render(false)
	if !at(nz, 10) || !at(nz, 28) || !at(nz, 50) {
		t.Error("nonzero: the target isn't a filled disc")
	}
	// A hole already drawn the other way round stays a hole.
	ring := append(circle(c, 45, false), circle(c, 30, true)...)
	img := Render(image.Pt(100, 100), f32.AffineId(), []Shape{{Path: ring, Fill: color.NRGBA{A: 255}, EvenOdd: true}})
	if at(img, 50) || !at(img, 10) {
		t.Error("evenodd: an opposite-wound hole was filled")
	}
}

func cos(a float64) float64 { return math.Cos(a) }
func sin(a float64) float64 { return math.Sin(a) }
