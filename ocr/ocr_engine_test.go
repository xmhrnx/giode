//go:build (linux && !android) || windows

package ocr

import (
	"context"
	"image"
	"image/color"
	"image/draw"
	"strings"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// page draws lines of text, large, black on white.
func page(lines ...string) image.Image {
	const scale = 4
	small := image.NewRGBA(image.Rect(0, 0, 400, 30*len(lines)+20))
	draw.Draw(small, small.Bounds(), image.White, image.Point{}, draw.Src)
	d := font.Drawer{Dst: small, Src: image.Black, Face: basicfont.Face7x13}
	for i, l := range lines {
		d.Dot = fixed.P(10, 25+30*i)
		d.DrawString(l)
	}
	big := image.NewRGBA(image.Rect(0, 0, small.Rect.Dx()*scale, small.Rect.Dy()*scale))
	for y := 0; y < big.Rect.Dy(); y++ {
		for x := 0; x < big.Rect.Dx(); x++ {
			big.Set(x, y, small.At(x/scale, y/scale))
		}
	}
	return big
}

// TestRecognizeText reads rendered text with the system's engine:
// Tesseract on Linux, Windows.Media.Ocr on Windows.
func TestRecognizeText(t *testing.T) {
	if !Supported() {
		t.Skip("no OCR engine")
	}
	res, err := Recognize(context.Background(), page("HELLO GIODE WORLD", "SECOND LINE 42"), Options{Languages: []string{"en-US"}})
	if err != nil {
		t.Skip("no OCR engine:", err)
	}
	t.Logf("%q", res.Text)
	if len(res.Lines) != 2 || !strings.Contains(res.Text, "HELLO") || !strings.Contains(res.Text, "42") {
		t.Fatalf("read %+v", res)
	}
	l := res.Lines[0]
	if len(l.Words) != 3 || l.Words[0].Text != "HELLO" || l.Box.Dx() < 200 {
		t.Errorf("first line %+v", l)
	}
	if l.Words[1].Box.Min.X <= l.Words[0].Box.Max.X {
		t.Errorf("word boxes out of order: %v %v", l.Words[0].Box, l.Words[1].Box)
	}
	// A language that isn't installed falls back to English.
	if _, err := Recognize(context.Background(), page("FALLBACK"), Options{Languages: []string{"ja"}}); err != nil {
		t.Error(err)
	}
	_ = color.Black
}
