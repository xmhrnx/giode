package fonts

import (
	"strings"
	"sync"
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
	if got := Resolve(font.Font{Typeface: "Brand"}); !sameFace(got, faces[before].Font) {
		t.Errorf("Brand regular resolves to %+v, want the first file's face %+v", got, faces[before].Font)
	}
	if got := Resolve(font.Font{Typeface: "Brand", Weight: font.Bold, Style: font.Italic}); !sameFace(got, faces[before+1].Font) {
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
		if got := Resolve(c.ask); !sameFace(got, c.want) {
			t.Errorf("Resolve(%+v) = %+v, want %+v", c.ask, got, c.want)
		}
	}
	// Register forgets them, with the collection.
	UseGofont()
	if got := Resolve(font.Font{Typeface: "Inter"}); Primary(got) != "Inter" {
		t.Errorf("after Register, Resolve still maps Inter to %+v", got)
	}
}

// With no fonts registered, text gets the Go fonts rather than nothing.
func TestDefaultFonts(t *testing.T) {
	defer Register(nil)
	shaper.Store(nil)
	collection = nil
	defaultFonts = sync.Once{}
	if Shaper() == nil {
		t.Fatal("no shaper without registered fonts")
	}
	if len(Collection()) == 0 {
		t.Error("the default shaper has no fonts")
	}

	// RegisterFont first adds beside the Go fonts.
	shaper.Store(nil)
	collection = nil
	defaultFonts = sync.Once{}
	if err := RegisterFont(File{Family: "Brand", Data: goregular.TTF}); err != nil {
		t.Fatal(err)
	}
	if n := len(Collection()); n < 2 {
		t.Errorf("%d faces after RegisterFont, want the Go fonts and Brand", n)
	}
}

// SetDefaultFont gives text without a family the default family's
// faces, by weight; "" goes back to the Go fonts.
func TestSetDefaultFont(t *testing.T) {
	defer Register(nil)
	defer SetDefaultFont("")
	UseGofont()
	before := len(Collection())
	if err := RegisterFont(
		File{Family: "Inter", Data: goregular.TTF},
		File{Family: "Inter", Weight: "bold", Data: gobold.TTF},
	); err != nil {
		t.Fatal(err)
	}
	faces := Collection()
	old := Shaper()
	SetDefaultFont("Inter")
	if DefaultFont() != "Inter" {
		t.Errorf("DefaultFont %q", DefaultFont())
	}
	if Shaper() == old {
		t.Error("the shaper wasn't replaced, so text laid out in the old default stays")
	}
	if got := Resolve(font.Font{}); !sameFace(got, faces[before].Font) {
		t.Errorf("no family resolves to %+v, want Inter regular %+v", got, faces[before].Font)
	}
	if got := Resolve(font.Font{Weight: font.Bold}); !sameFace(got, faces[before+1].Font) {
		t.Errorf("no family, bold, resolves to %+v, want Inter bold %+v", got, faces[before+1].Font)
	}
	// A family set on the text still wins.
	if got := Resolve(font.Font{Typeface: "Go Mono"}); Primary(got) != "Go Mono" {
		t.Errorf("an explicit family resolves to %+v", got)
	}
	SetDefaultFont("")
	// The Go fonts, then the emoji fonts after them.
	if got := Resolve(font.Font{}); Primary(got) != "Go" || !strings.Contains(string(got.Typeface), "Noto Color Emoji") {
		t.Errorf("after SetDefaultFont(\"\"), no family resolves to %+v, want the Go fonts' default", got)
	}
}

// sameFace compares the face Resolve picked: its primary family, weight
// and style (the family list goes on with the fallbacks).
func sameFace(a, b font.Font) bool {
	return Primary(a) == Primary(b) && a.Weight == b.Weight && a.Style == b.Style
}

// A family's missing characters come from the app's other registered
// fonts, the default first, before the system's.
func TestFallbackFamilies(t *testing.T) {
	defer Register(nil)
	defer SetDefaultFont("")
	UseGofont()
	if err := RegisterFont(File{Family: "Book", Data: goregular.TTF}); err != nil {
		t.Fatal(err)
	}
	SetDefaultFont("Book")
	f := Resolve(font.Font{Typeface: "Go Mono"})
	if got := string(f.Typeface); !strings.HasPrefix(got, `"Go Mono", "Go"`) || strings.Count(got, `"Go Mono"`) != 1 {
		t.Errorf("Go Mono's family list %s: want Go Mono, then the default (Book's file, Go), once each", got)
	}
	if got := Resolve(font.Font{}); Primary(got) != "Go" {
		t.Errorf("no family: %s", got.Typeface)
	}
}
