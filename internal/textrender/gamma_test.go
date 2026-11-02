package textrender

import (
	"image/color"
	"math"
	"testing"

	"golang.org/x/image/math/fixed"

	"github.com/xmhrnx/giode/fonts"
)

func TestCoverageLUT(t *testing.T) {
	dark := coverageLUT(color.NRGBA{R: 0x20, G: 0x20, B: 0x20, A: 0xff})
	light := coverageLUT(color.NRGBA{R: 0xf0, G: 0xf0, B: 0xf0, A: 0xff})
	for _, l := range []*[256]uint8{dark, light} {
		if l[0] != 0 || l[255] != 255 {
			t.Errorf("ends %d, %d; want 0, 255", l[0], l[255])
		}
		for i := 1; i < 256; i++ {
			if l[i] < l[i-1] {
				t.Fatalf("not monotonic at %d", i)
			}
		}
	}
	// Dark text gets more coverage, light text less, than linear.
	if dark[128] <= 128 || light[128] >= 128 {
		t.Errorf("half coverage: dark %d, light %d", dark[128], light[128])
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
