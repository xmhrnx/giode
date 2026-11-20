//go:build windows

package systray

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/png"
	"testing"
)

func TestPNGToICO(t *testing.T) {
	p := testPNG(t)
	ico := pngToICO(p)
	if len(ico) != 22+len(p) {
		t.Fatalf("ico is %d bytes, want %d", len(ico), 22+len(p))
	}
	var hdr [3]uint16
	binary.Read(bytes.NewReader(ico), binary.LittleEndian, &hdr)
	if hdr != [3]uint16{0, 1, 1} || ico[6] != 32 || ico[7] != 32 || !bytes.Equal(ico[22:], p) {
		t.Errorf("bad header %v %d %d", hdr, ico[6], ico[7])
	}
	if pngToICO([]byte("not a png")) != nil {
		t.Error("garbage converted")
	}
}

func testPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
