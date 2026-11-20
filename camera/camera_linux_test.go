//go:build linux && !android

package camera

import "testing"

func TestYUYV(t *testing.T) {
	// Two pixels: white and black share chroma 128.
	src := []byte{235, 128, 16, 128}
	dst := make([]byte, 8)
	yuyvToBGRA(dst, src, 2, 1, 4)
	if dst[0] < 250 || dst[1] < 250 || dst[2] < 250 || dst[4] > 5 || dst[3] != 255 {
		t.Errorf("white and black: %v", dst)
	}
	// Pure red in BT.601: Y 81, U 90, V 240.
	yuyvToBGRA(dst, []byte{81, 90, 81, 240}, 2, 1, 4)
	if dst[2] < 240 || dst[1] > 15 || dst[0] > 15 {
		t.Errorf("red: %v", dst[:4])
	}
}
