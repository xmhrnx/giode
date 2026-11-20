//go:build windows

package filedrag

import (
	"encoding/binary"
	"testing"
	"unicode/utf16"
)

// dropFiles lays paths out as Explorer reads a CF_HDROP: the DROPFILES
// header, then NUL-ended UTF-16 names and a final NUL.
func TestDropFiles(t *testing.T) {
	b := dropFiles([]string{`C:\a.txt`, `D:\ü.pdf`})
	if off := binary.LittleEndian.Uint32(b); off != 20 {
		t.Fatalf("pFiles = %d, want 20", off)
	}
	if wide := binary.LittleEndian.Uint32(b[16:]); wide != 1 {
		t.Fatalf("fWide = %d, want 1", wide)
	}
	var u []uint16
	for i := 20; i+1 < len(b); i += 2 {
		u = append(u, binary.LittleEndian.Uint16(b[i:]))
	}
	want := append(append(append(utf16.Encode([]rune(`C:\a.txt`)), 0), utf16.Encode([]rune(`D:\ü.pdf`))...), 0, 0)
	if string(utf16.Decode(u)) != string(utf16.Decode(want)) {
		t.Errorf("names %q, want %q", string(utf16.Decode(u)), string(utf16.Decode(want)))
	}
	if z := utf16z("hé"); len(z) != 6 || z[4] != 0 || z[5] != 0 {
		t.Errorf("utf16z = %v", z)
	}
}
