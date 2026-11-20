package rtc

import (
	"bytes"
	"image"
	"image/color"
	"math"
	"testing"

	"golang.org/x/image/vp8"
)

// testPicture has gradients, hard edges and a circle, at an odd size.
func testPicture(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := color.RGBA{uint8(x * 255 / w), uint8(y * 255 / h), 128, 255}
			if x > w/2 && y < h/3 {
				c = color.RGBA{220, 30, 40, 255}
			}
			dx, dy := x-w/4, y-2*h/3
			if dx*dx+dy*dy < (h/6)*(h/6) {
				c = color.RGBA{250, 250, 250, 255}
			}
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func decodeVP8(t *testing.T, b []byte) *image.YCbCr {
	t.Helper()
	d := vp8.NewDecoder()
	d.Init(bytes.NewReader(b), len(b))
	fh, err := d.DecodeFrameHeader()
	if err != nil {
		t.Fatal(err)
	}
	if !fh.KeyFrame {
		t.Fatal("not a key frame")
	}
	img, err := d.DecodeFrame()
	if err != nil {
		t.Fatal(err)
	}
	return img
}

// psnr compares the luma the encoder fed in with the decoded luma.
func psnr(a []byte, as int, b *image.YCbCr, w, h int) float64 {
	var se float64
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			d := float64(a[y*as+x]) - float64(b.Y[y*b.YStride+x])
			se += d * d
		}
	}
	mse := se / float64(w*h)
	if mse == 0 {
		return 99
	}
	return 10 * math.Log10(255*255/mse)
}

func TestVP8EncoderDecodes(t *testing.T) {
	for _, tc := range []struct{ w, h, q int }{{64, 48, 10}, {177, 99, 30}, {320, 240, 60}, {33, 17, 127}, {16, 16, 0}} {
		e := newVP8Encoder(tc.w, tc.h, tc.q)
		src := testPicture(tc.w, tc.h)
		b := e.Encode(src)
		img := decodeVP8(t, b)
		if img.Rect.Dx() != tc.w || img.Rect.Dy() != tc.h {
			t.Fatalf("%v: decoded %v", tc, img.Rect)
		}
		// The decoder must rebuild exactly what the encoder predicted
		// from.
		for y := 0; y < tc.h; y++ {
			for x := 0; x < tc.w; x++ {
				if img.Y[y*img.YStride+x] != e.ry[y*e.ys+x] {
					t.Fatalf("%v: luma (%d, %d) is %d, the encoder had %d", tc, x, y, img.Y[y*img.YStride+x], e.ry[y*e.ys+x])
				}
			}
		}
		for y := 0; y < (tc.h+1)/2; y++ {
			for x := 0; x < (tc.w+1)/2; x++ {
				if img.Cb[y*img.CStride+x] != e.ru[y*e.cs+x] || img.Cr[y*img.CStride+x] != e.rv[y*e.cs+x] {
					t.Fatalf("%v: chroma (%d, %d) differs", tc, x, y)
				}
			}
		}
		p := psnr(e.sy, e.ys, img, tc.w, tc.h)
		t.Logf("%dx%d q%d: %d bytes, %.1f dB", tc.w, tc.h, tc.q, len(b), p)
		if tc.q <= 30 && p < 35 {
			t.Errorf("%v: %.1f dB", tc, p)
		}
	}
}

func TestVP8Codec(t *testing.T) {
	enc, _ := VP8.NewEncoder(96, 64, 30)
	dec, _ := VP8.NewDecoder()
	src := testPicture(96, 64)
	b, err := enc.Encode(src, true)
	if err != nil {
		t.Fatal(err)
	}
	img, err := dec.Decode(b)
	if err != nil || img == nil {
		t.Fatal(img, err)
	}
	got := img.(*image.RGBA).RGBAAt(80, 5) // the red corner
	if got.R < 190 || got.G > 70 || got.B > 80 {
		t.Errorf("red came back as %v", got)
	}
	// Not a key frame: wait for one.
	if img, err := dec.Decode([]byte{1, 0, 0}); img != nil || err != nil {
		t.Errorf("inter frame: %v, %v", img, err)
	}
}
