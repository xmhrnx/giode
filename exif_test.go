package giode

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

// withOrientation inserts an EXIF APP1 segment with orientation o after
// a JPEG's SOI marker.
func withOrientation(jpg []byte, o uint16, bo binary.ByteOrder) []byte {
	var t bytes.Buffer
	if bo == binary.LittleEndian {
		t.WriteString("II")
	} else {
		t.WriteString("MM")
	}
	binary.Write(&t, bo, uint16(42))
	binary.Write(&t, bo, uint32(8))
	binary.Write(&t, bo, uint16(1))
	binary.Write(&t, bo, uint16(0x0112))
	binary.Write(&t, bo, uint16(3)) // SHORT
	binary.Write(&t, bo, uint32(1))
	binary.Write(&t, bo, o)
	binary.Write(&t, bo, uint16(0))
	binary.Write(&t, bo, uint32(0))
	seg := append([]byte("Exif\x00\x00"), t.Bytes()...)
	out := []byte{0xFF, 0xD8, 0xFF, 0xE1}
	out = binary.BigEndian.AppendUint16(out, uint16(len(seg)+2))
	out = append(out, seg...)
	return append(out, jpg[2:]...)
}

func TestJPEGOrientation(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 40, 20))
	var buf bytes.Buffer
	jpeg.Encode(&buf, img, nil)
	if o := jpegOrientation(buf.Bytes()); o != 1 {
		t.Errorf("no EXIF: orientation %d", o)
	}
	for _, bo := range []binary.ByteOrder{binary.LittleEndian, binary.BigEndian} {
		if o := jpegOrientation(withOrientation(buf.Bytes(), 6, bo)); o != 6 {
			t.Errorf("%v: orientation %d, want 6", bo, o)
		}
	}
}

func TestToJPEGAppliesOrientation(t *testing.T) {
	// A wide image with a red left half, tagged "rotate 90 clockwise".
	img := image.NewRGBA(image.Rect(0, 0, 40, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 40; x++ {
			c := color.RGBA{0, 0, 255, 255}
			if x < 20 {
				c = color.RGBA{255, 0, 0, 255}
			}
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95})
	out, err := toJPEG(bytes.NewReader(withOrientation(buf.Bytes(), 6, binary.BigEndian)), 0)
	if err != nil {
		t.Fatal(err)
	}
	got, err := jpeg.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	if b := got.Bounds(); b.Dx() != 20 || b.Dy() != 40 {
		t.Fatalf("upright size %v, want 20x40", b.Size())
	}
	// Turned clockwise, the red left half is now on top.
	if r, _, b, _ := got.At(10, 5).RGBA(); r>>8 < 200 || b>>8 > 60 {
		t.Errorf("top isn't red: r=%d b=%d", r>>8, b>>8)
	}
	if r, _, b, _ := got.At(10, 35).RGBA(); b>>8 < 200 || r>>8 > 60 {
		t.Errorf("bottom isn't blue: r=%d b=%d", r>>8, b>>8)
	}
}

func TestShareFilesMIME(t *testing.T) {
	for _, tc := range []struct {
		paths []string
		want  string
	}{
		{[]string{"a.jpg", "b.jpg"}, "image/jpeg"},
		{[]string{"a.jpg", "b.png"}, "image/*"},
		{[]string{"a.jpg", "b.pdf"}, "*/*"},
	} {
		if got := (ShareFiles{Paths: tc.paths}).mimeType(); got != tc.want {
			t.Errorf("%v: %s, want %s", tc.paths, got, tc.want)
		}
	}
}
