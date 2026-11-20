package screencap

import (
	"image"
	"testing"
)

func TestFromBGRAScales(t *testing.T) {
	c := &Capture{opts: Options{MaxSize: 2}}
	// 4×2 BGRA, stride 20: left half blue, right half red.
	src := make([]byte, 20*2)
	for y := 0; y < 2; y++ {
		for x := 0; x < 4; x++ {
			o := y*20 + x*4
			if x < 2 {
				src[o] = 255 // B
			} else {
				src[o+2] = 255 // R
			}
		}
	}
	img := c.fromBGRA(src, 4, 2, 20)
	if img.Bounds() != image.Rect(0, 0, 2, 1) {
		t.Fatalf("bounds %v", img.Bounds())
	}
	if got := img.RGBAAt(0, 0); got.B != 255 || got.R != 0 || got.A != 255 {
		t.Errorf("left %v", got)
	}
	if got := img.RGBAAt(1, 0); got.R != 255 || got.B != 0 {
		t.Errorf("right %v", got)
	}
}

func TestFromNV12(t *testing.T) {
	c := &Capture{opts: Options{MaxSize: -1}}
	// 2×2 mid grey, video range: Y 126, U V 128.
	y := []byte{126, 126, 126, 126}
	uv := []byte{128, 128}
	img := c.fromNV12(y, 2, uv, 2, 2, 2, false)
	if got := img.RGBAAt(1, 1); got.R < 120 || got.R > 135 || got.R != got.G || got.G != got.B {
		t.Errorf("grey = %v", got)
	}
	// Full range pure red: Y 76, U 85, V 255 (BT.709 approximately).
	img = c.fromNV12([]byte{54, 54, 54, 54}, 2, []byte{99, 255}, 2, 2, 2, true)
	if got := img.RGBAAt(0, 0); got.R < 230 || got.G > 30 || got.B > 30 {
		t.Errorf("red = %v", got)
	}
}

func TestFromRGBAOpaque(t *testing.T) {
	src := []byte{1, 2, 3, 0, 4, 5, 6, 0, 9, 9, 7, 8, 9, 0, 1, 2, 3, 0, 9, 9}
	img := fromRGBA(src, 2, 2, 10)
	if got := img.RGBAAt(1, 1); got.R != 1 || got.G != 2 || got.B != 3 || got.A != 255 {
		t.Errorf("pixel = %v", got)
	}
}
