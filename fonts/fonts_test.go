package fonts

import (
	"testing"

	"gioui.org/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goitalic"
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
	// Text asks for "Brand" by weight and style; each face keeps its
	// file's own name.
	if got := Resolve(font.Font{Typeface: "Brand"}); got != faces[before].Font {
		t.Errorf("Brand regular resolves to %+v, want the first file's face %+v", got, faces[before].Font)
	}
	if got := Resolve(font.Font{Typeface: "Brand", Weight: font.Bold, Style: font.Italic}); got != faces[before+1].Font {
		t.Errorf("Brand bold italic resolves to %+v, want the second file's face %+v", got, faces[before+1].Font)
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

// Resolve maps a family registered with RegisterFont to its faces by
// weight and style: the same style and the closest weight, the heavier
// on a tie.
func TestResolve(t *testing.T) {
	defer UseGofont()
	Register(nil)
	if err := RegisterFont(
		File{Family: "Inter", Data: goregular.TTF},
		File{Family: "Inter", Weight: "bold", Data: gobold.TTF},
		File{Family: "Inter", Italic: true, Data: goitalic.TTF},
	); err != nil {
		t.Fatal(err)
	}
	regular, bold, italic := Collection()[0].Font, Collection()[1].Font, Collection()[2].Font
	cases := []struct {
		ask  font.Font
		want font.Font
	}{
		{font.Font{Typeface: "Inter"}, regular},
		{font.Font{Typeface: "Inter", Weight: font.Bold}, bold},
		{font.Font{Typeface: "Inter", Weight: font.SemiBold}, bold},
		{font.Font{Typeface: "Inter", Weight: font.Light}, regular},
		{font.Font{Typeface: "Inter", Style: font.Italic}, italic},
		{font.Font{Typeface: "Other", Weight: font.Bold}, font.Font{Typeface: "Other", Weight: font.Bold}},
	}
	for _, c := range cases {
		if got := Resolve(c.ask); got != c.want {
			t.Errorf("Resolve(%+v) = %+v, want %+v", c.ask, got, c.want)
		}
	}
	// Register forgets them, with the collection.
	UseGofont()
	if got := Resolve(font.Font{Typeface: "Inter"}); got.Typeface != "Inter" {
		t.Errorf("after Register, Resolve still maps Inter to %+v", got)
	}
}
