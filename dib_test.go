package giode

import (
	"encoding/binary"
	"image/color"
	"testing"
)

func dibHeader(size, w, h, bpp, compression int) []byte {
	b := make([]byte, size)
	le := binary.LittleEndian
	le.PutUint32(b[0:], uint32(size))
	le.PutUint32(b[4:], uint32(int32(w)))
	le.PutUint32(b[8:], uint32(int32(h)))
	le.PutUint16(b[12:], 1)
	le.PutUint16(b[14:], uint16(bpp))
	le.PutUint32(b[16:], uint32(compression))
	return b
}

func TestParseDIB(t *testing.T) {
	// 2x2, 24 bit, bottom-up: rows padded to 8 bytes.
	b := dibHeader(40, 2, 2, 24, 0)
	b = append(b,
		0, 0, 255 /* red */, 0, 255, 0 /* green */, 0, 0, // bottom row
		255, 0, 0 /* blue */, 255, 255, 255 /* white */, 0, 0, // top row
	)
	img, err := parseDIB(b)
	if err != nil {
		t.Fatal(err)
	}
	want := map[[2]int]color.NRGBA{
		{0, 0}: {0, 0, 255, 255}, {1, 0}: {255, 255, 255, 255},
		{0, 1}: {255, 0, 0, 255}, {1, 1}: {0, 255, 0, 255},
	}
	for p, c := range want {
		if got := color.NRGBAModel.Convert(img.At(p[0], p[1])).(color.NRGBA); got != c {
			t.Errorf("24-bit (%d,%d) = %v, want %v", p[0], p[1], got, c)
		}
	}

	// 1x2, 32-bit V5 top-down with an alpha mask.
	v5 := dibHeader(124, 1, -2, 32, 3)
	le := binary.LittleEndian
	le.PutUint32(v5[40:], 0x00ff0000)
	le.PutUint32(v5[44:], 0x0000ff00)
	le.PutUint32(v5[48:], 0x000000ff)
	le.PutUint32(v5[52:], 0xff000000)
	v5 = append(v5, 0x10, 0x20, 0x30, 0x80, 0, 0, 0xff, 0xff)
	img, err = parseDIB(v5)
	if err != nil {
		t.Fatal(err)
	}
	if got := img.At(0, 0).(color.NRGBA); got != (color.NRGBA{0x30, 0x20, 0x10, 0x80}) {
		t.Errorf("32-bit top pixel %v", got)
	}
	if got := img.At(0, 1).(color.NRGBA); got != (color.NRGBA{0xff, 0, 0, 0xff}) {
		t.Errorf("32-bit bottom pixel %v", got)
	}

	// 32-bit BI_RGB with an unused alpha byte: opaque.
	x := dibHeader(40, 1, 1, 32, 0)
	x = append(x, 1, 2, 3, 0)
	img, _ = parseDIB(x)
	if got := img.At(0, 0).(color.NRGBA); got.A != 255 {
		t.Errorf("unused alpha: %v, want opaque", got)
	}

	if _, err := parseDIB(dibHeader(40, 1, 1, 8, 0)); err == nil {
		t.Error("8-bit palette bitmap accepted")
	}
	if _, err := parseDIB([]byte{1, 2, 3}); err == nil {
		t.Error("garbage accepted")
	}
}
