package camera

import (
	"image"
	"testing"

	"gioui.org/f32"
)

// testFrame is a 4x2 YUV frame whose brightness is the pixel index, with
// neutral colour, so each pixel's grey value names where it came from.
func testFrame(rotation int, mirror bool) rawFrame {
	w, h := 4, 2
	y := make([]byte, w*h)
	for i := range y {
		y[i] = byte(10 * (i + 1))
	}
	uv := make([]byte, w*h/2)
	for i := range uv {
		uv[i] = 128
	}
	return rawFrame{y: y, u: uv, v: uv, w: w, h: h, yStride: w, uvStride: w, uvPixStride: 2, rotation: rotation, mirror: mirror}
}

// grid reads an RGBA image's red channel as rows of source indices.
func grid(img *image.RGBA) [][]int {
	var rows [][]int
	for y := 0; y < img.Rect.Dy(); y++ {
		var row []int
		for x := 0; x < img.Rect.Dx(); x++ {
			row = append(row, int(img.Pix[y*img.Stride+x*4])/10-1)
		}
		rows = append(rows, row)
	}
	return rows
}

func TestYUVRotation(t *testing.T) {
	// The source, by index:
	//   0 1 2 3
	//   4 5 6 7
	for _, tc := range []struct {
		rot    int
		mirror bool
		want   [][]int
	}{
		{0, false, [][]int{{0, 1, 2, 3}, {4, 5, 6, 7}}},
		{90, false, [][]int{{4, 0}, {5, 1}, {6, 2}, {7, 3}}},
		{180, false, [][]int{{7, 6, 5, 4}, {3, 2, 1, 0}}},
		{270, false, [][]int{{3, 7}, {2, 6}, {1, 5}, {0, 4}}},
		{0, true, [][]int{{3, 2, 1, 0}, {7, 6, 5, 4}}},
		{90, true, [][]int{{0, 4}, {1, 5}, {2, 6}, {3, 7}}},
		{270, true, [][]int{{7, 3}, {6, 2}, {5, 1}, {4, 0}}},
	} {
		got := grid(yuvToRGBA(nil, testFrame(tc.rot, tc.mirror)))
		if !equal(got, tc.want) {
			t.Errorf("rotation %d mirror %v: %v, want %v", tc.rot, tc.mirror, got, tc.want)
		}
		g := grayFromY(testFrame(tc.rot, tc.mirror), 0)
		for y, row := range tc.want {
			for x, idx := range row {
				if int(g.Pix[y*g.Stride+x]) != 10*(idx+1) {
					t.Errorf("rotation %d mirror %v: grey (%d,%d) = %d, want %d", tc.rot, tc.mirror, x, y, g.Pix[y*g.Stride+x], 10*(idx+1))
				}
			}
		}
	}
}

func TestYUVColour(t *testing.T) {
	// Pure red in BT.601 full range: Y 76, Cb 85, Cr 255.
	f := rawFrame{y: []byte{76, 76, 76, 76}, u: []byte{85}, v: []byte{255}, w: 2, h: 2, yStride: 2, uvStride: 2, uvPixStride: 1}
	img := yuvToRGBA(nil, f)
	r, g, b := img.Pix[0], img.Pix[1], img.Pix[2]
	if r < 250 || g > 5 || b > 5 {
		t.Errorf("red came out as (%d, %d, %d)", r, g, b)
	}
}

func TestBGRA(t *testing.T) {
	bgra := []byte{1, 2, 3, 255, 4, 5, 6, 255, 0, 0, 0, 0} // stride 12 for 2 pixels
	img := bgraToRGBA(nil, bgra, 2, 1, 12)
	if got := img.Pix[:8]; got[0] != 3 || got[1] != 2 || got[2] != 1 || got[4] != 6 || got[6] != 4 {
		t.Errorf("BGRA to RGBA: %v", got)
	}
	g := grayFromRGBA(img, 0)
	if g.Rect.Dx() != 2 || g.Pix[0] == 0 {
		t.Errorf("grey from RGBA: %v", g.Pix)
	}
}

func TestAnalysisSize(t *testing.T) {
	for _, tc := range []struct{ w, h, long, gw, gh int }{
		{1280, 960, 640, 640, 480},
		{960, 1280, 640, 480, 640},
		{320, 240, 640, 320, 240},
	} {
		if gw, gh := analysisSize(tc.w, tc.h, tc.long); gw != tc.gw || gh != tc.gh {
			t.Errorf("analysisSize(%d,%d,%d) = %d,%d, want %d,%d", tc.w, tc.h, tc.long, gw, gh, tc.gw, tc.gh)
		}
	}
	f := testFrame(90, false)
	f.y = make([]byte, 1280*960)
	f.u = make([]byte, 1280*960/2)
	f.v = f.u
	f.w, f.h, f.yStride, f.uvStride = 1280, 960, 1280, 1280
	if g := grayFromY(f, 640); g.Rect.Dx() != 480 || g.Rect.Dy() != 640 {
		t.Errorf("a rotated 1280x960 frame analysed at %v, want 480x640", g.Rect.Size())
	}
}

func TestFitFrame(t *testing.T) {
	// A 3:4 frame in a 300x300 view.
	if got := fitFrame(image.Pt(300, 400), image.Pt(300, 300), false); got != (frameRect{Min: f32.Pt(37.5, 0), W: 225, H: 300}) {
		t.Errorf("fit: %v", got)
	}
	if got := fitFrame(image.Pt(300, 400), image.Pt(300, 300), true); got != (frameRect{Min: f32.Pt(0, -50), W: 300, H: 400}) {
		t.Errorf("fill: %v", got)
	}
}

func equal(a, b [][]int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if len(a[i]) != len(b[i]) {
			return false
		}
		for j := range a[i] {
			if a[i][j] != b[i][j] {
				return false
			}
		}
	}
	return true
}
