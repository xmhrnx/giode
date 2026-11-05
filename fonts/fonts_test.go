package fonts

import (
	"testing"

	"gioui.org/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

func TestRegisterFont(t *testing.T) {
	defer Register(nil)
	UseGofont()
	before := len(Collection())
	err := RegisterFont(
		File{Family: "Brand", Data: goregular.TTF},
		File{Family: "Brand", Weight: "bold", Italic: true, Data: gobold.TTF},
	)
	if err != nil {
		t.Fatal(err)
	}
	faces := Collection()
	if len(faces) != before+2 {
		t.Fatalf("%d faces, want %d", len(faces), before+2)
	}
	reg, bold := faces[before].Font, faces[before+1].Font
	if reg.Typeface != "Brand" || reg.Weight != font.Normal || reg.Style != font.Regular {
		t.Errorf("regular face %+v", reg)
	}
	if bold.Typeface != "Brand" || bold.Weight != font.Bold || bold.Style != font.Italic {
		t.Errorf("bold italic face %+v", bold)
	}
	if Shaper() == nil {
		t.Error("no shaper")
	}
	if err := RegisterFont(File{Family: "Bad", Data: []byte("not a font")}); err == nil {
		t.Error("garbage registered")
	}
	if len(Collection()) != before+2 {
		t.Error("a failed registration changed the fonts")
	}
}
