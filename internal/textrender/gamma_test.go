package textrender

import (
	"image/color"
	"math"
	"testing"

	"golang.org/x/image/math/fixed"

	"github.com/xmhrnx/giode/fonts"
)

func TestCoverageLUT(t *testing.T) {
	white := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	dark := coverageLUT(color.NRGBA{R: 0x20, G: 0x20, B: 0x20, A: 0xff}, white)
	light := coverageLUT(color.NRGBA{R: 0xf0, G: 0xf0, B: 0xf0, A: 0xff}, color.NRGBA{R: 0x10, G: 0x10, B: 0x10, A: 0xff})
	// Gray text on white is dark text there; white text on green is
	// light text, though less than over black.
	gray := coverageLUT(color.NRGBA{R: 0xc2, G: 0xc5, B: 0xcc, A: 0xff}, white)
	onGreen := coverageLUT(white, color.NRGBA{R: 0x10, G: 0xb9, B: 0x81, A: 0xff})
	for _, l := range []*[256]uint8{dark, light, gray, onGreen} {
		if l[0] != 0 || l[255] != 255 {
			t.Errorf("ends %d, %d; want 0, 255", l[0], l[255])
		}
		for i := 1; i < 256; i++ {
			if l[i] < l[i-1] {
				t.Fatalf("not monotonic at %d", i)
			}
		}
	}
	if dark[128] <= 128 || gray[128] <= 128 || light[128] >= 128 || onGreen[128] >= 128 || onGreen[128] <= light[128] {
		t.Errorf("half coverage: dark %d, gray %d, light %d, on green %d", dark[128], gray[128], light[128], onGreen[128])
	}
	// The same colors: no correction.
	if same := coverageLUT(white, white); same[128] != 128 {
		t.Errorf("no contrast: %d", same[128])
	}
}

// Boxes push their backgrounds; translucent ones over what's below.
func TestBackgroundStack(t *testing.T) {
	DefaultBackground = func() color.NRGBA { return color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff} }
	pop := PushBackground(color.NRGBA{R: 0, G: 0, B: 0, A: 0xff})
	pop2 := PushBackground(color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x80})
	if b := Background(); b.R != 0x80 {
		t.Errorf("half white over black: %v", b)
	}
	pop2()
	pop()
	if b := Background(); b.R != 0xff {
		t.Errorf("default: %v", b)
	}
}

func TestGridFit(t *testing.T) {
	fonts.UseGofont()
	fs := faces(fonts.Shaper())
	if len(fs) == 0 {
		t.Skip("no faces")
	}
	face := fs[0]
	for _, px := range []int{11, 13, 15} {
		scale := float32(px) / float32(face.Upem())
		xh := xHeight(face) * scale * gridFit(face, 0, fixed.I(px), scale)
		if math.Abs(float64(xh)-math.Round(float64(xh))) > 1e-3 {
			t.Errorf("%dpx: x-height %.3f isn't whole pixels", px, xh)
		}
	}
	if gridFit(face, 0, fixed.I(40), float32(40)/float32(face.Upem())) != 1 {
		t.Error("large text was fitted")
	}
}
