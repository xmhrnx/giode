package linear

import (
	"image"
	"image/color"
	"testing"
)

// Half-covered white is half the light: 188 in sRGB, not 128.
func TestPremul(t *testing.T) {
	if v := Premul(255, 128); v < 186 || v > 189 {
		t.Errorf("white at half alpha: %d", v)
	}
	if Premul(0, 128) != 0 || Premul(255, 255) != 255 {
		t.Error("ends")
	}
	n := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	n.SetNRGBA(0, 0, color.NRGBA{255, 255, 255, 128})
	n.SetNRGBA(1, 0, color.NRGBA{10, 20, 30, 255})
	out := Image(n).(*image.RGBA)
	if p := out.RGBAAt(0, 0); p.R < 186 || p.A != 128 {
		t.Errorf("converted %v", p)
	}
	if p := out.RGBAAt(1, 0); p != (color.RGBA{10, 20, 30, 255}) {
		t.Errorf("opaque pixel changed: %v", p)
	}
	r := image.NewRGBA(image.Rect(0, 0, 1, 1))
	r.SetRGBA(0, 0, color.RGBA{128, 128, 128, 128}) // white at half, sRGB-premultiplied
	FixRGBA(r)
	if p := r.RGBAAt(0, 0); p.R < 186 || p.R > 189 {
		t.Errorf("fixed %v", p)
	}
	opaque := image.NewRGBA(image.Rect(0, 0, 1, 1))
	opaque.SetRGBA(0, 0, color.RGBA{1, 2, 3, 255})
	if Image(opaque) != image.Image(opaque) {
		t.Error("an opaque image was copied")
	}
}
