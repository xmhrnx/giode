// Package fonts holds the text shaper shared by all Giode elements.
//
// Text uses the Go fonts unless the app registers its own (Register,
// RegisterFont); characters a font lacks fall back to the system's
// fonts where the platform has them.
package fonts

import (
	"fmt"
	"sync"
	"sync/atomic"

	"gioui.org/font"
	"gioui.org/font/gofont"
	"gioui.org/font/opentype"
	"gioui.org/text"

	"github.com/xmhrnx/giode/styles/properties"
)

var (
	shaper atomic.Pointer[text.Shaper]
	// shaped counts the bytes of text shaped since the shaper was made.
	shaped atomic.Int64

	mu         sync.Mutex
	collection []font.FontFace
)

// Register replaces the font collection used to shape text.
func Register(faces []font.FontFace) {
	register(faces, nil)
}

// register replaces the collection, and the RegisterFont aliases with
// as.
func register(faces []font.FontFace, as []alias) {
	mu.Lock()
	collection, aliases = faces, as
	mu.Unlock()
	shaper.Store(text.NewShaper(text.WithCollection(faces)))
	shaped.Store(0)
}

// Collection returns the registered font collection, in the order the
// shaper loaded it.
func Collection() []font.FontFace {
	mu.Lock()
	defer mu.Unlock()
	return collection
}

// Shaped records that n bytes of text were laid out with the shaper, for
// Refresh.
func Shaped(n int) {
	shaped.Add(int64(n))
}

// Refresh replaces the shaper with a fresh one over the same fonts once
// more than limit bytes of text were shaped with it, and reports whether
// it did. Gio's shaper keeps the last thousand paragraphs it laid out,
// however long - in a long chat, a hundred megabytes and more, kept
// after the chat is gone - and Giode keeps its own cache of laid out
// text, so letting Gio's go costs little: a new shaper over loaded
// fonts takes well under a millisecond. Call it between frames: text
// laid out from then on uses the new shaper.
func Refresh(limit int) bool {
	if shaped.Load() <= int64(limit) {
		return false
	}
	mu.Lock()
	faces := collection
	mu.Unlock()
	if shaper.Load() == nil {
		return false
	}
	shaper.Store(text.NewShaper(text.WithCollection(faces)))
	shaped.Store(0)
	return true
}

// UseGofont uses the Go fonts, embedded in the binary, for text. They're
// the default; call it to go back to them after Register.
func UseGofont() {
	Register(gofont.Collection())
}

// Shaper returns the text shaper. With no fonts registered it loads the
// Go fonts, so text shows without any setup.
func Shaper() *text.Shaper {
	if s := shaper.Load(); s != nil {
		return s
	}
	defaultFonts.Do(func() {
		if shaper.Load() == nil {
			UseGofont()
		}
	})
	return shaper.Load()
}

// defaultFonts loads the Go fonts once when text is shaped before any
// fonts were registered.
var defaultFonts sync.Once

// File is a font file to register with RegisterFont: the TTF, OTF or
// TTC data, and the family, weight and style Styles select it by
// (FontFamily, FontWeight, FontStyle "italic").
type File struct {
	Family string
	// Weight is a CSS weight name or number ("bold", "600"); empty
	// is normal.
	Weight properties.FontWeight
	Italic bool
	Data   []byte
}

// RegisterFont adds font files to the fonts used to shape text, beside
// any registered before, or beside the default Go fonts, which text in
// families you don't register falls back to:
//
//	//go:embed fonts/Inter-Regular.ttf
//	var interRegular []byte
//	//go:embed fonts/Inter-Bold.ttf
//	var interBold []byte
//
//	fonts.RegisterFont(
//		fonts.File{Family: "Inter", Data: interRegular},
//		fonts.File{Family: "Inter", Weight: "bold", Data: interBold},
//	)
//
// Text styled FontFamily: "Inter" then uses them, with the closest
// registered weight and style. The family, weight and style given here
// are what Styles select a file by, whatever the file itself says. A
// collection (TTC) registers all its faces.
func RegisterFont(files ...File) error {
	faces := append([]font.FontFace(nil), Collection()...)
	if shaper.Load() == nil {
		// Beside the default Go fonts.
		faces = gofont.Collection()
	}
	var added []alias
	for _, f := range files {
		parsed, err := opentype.ParseCollection(f.Data)
		if err != nil {
			return fmt.Errorf("giode: font %q: %w", f.Family, err)
		}
		style := font.Regular
		if f.Italic {
			style = font.Italic
		}
		for _, face := range parsed {
			// The face keeps the family and weight its file gives: Gio
			// resolves a face renamed to another family to a copy of
			// it, numbered apart from the registered fonts and drawn
			// as shapes. Text asks for it by the name given here, which
			// Resolve maps to the file's own.
			added = append(added, alias{
				family: font.Typeface(f.Family), weight: f.Weight.Weight(), style: style,
				font: face.Font,
			})
			faces = append(faces, face)
		}
	}
	mu.Lock()
	as := append(append([]alias(nil), aliases...), added...)
	mu.Unlock()
	register(faces, as)
	return nil
}

// alias is a face registered with RegisterFont: the family, weight and
// style text asks for it by, and the face's own.
type alias struct {
	family font.Typeface
	weight font.Weight
	style  font.Style
	font   font.Font
}

var aliases []alias

// defaultFamily is the family text without a FontFamily uses; empty
// for the first registered face (the Go fonts' by default).
var defaultFamily font.Typeface

// SetDefaultFont makes family the font of all text that doesn't set a
// FontFamily - Giode's own components included (title bars, buttons,
// text fields, menus, toasts) - in place of the Go fonts. Register the
// family's files first (RegisterFont); "" goes back to the Go fonts.
// Call it before the window opens: text laid out from then on uses it.
func SetDefaultFont(family string) {
	mu.Lock()
	defaultFamily = font.Typeface(family)
	faces, as := collection, aliases
	mu.Unlock()
	if faces != nil {
		// A fresh shaper drops text laid out in the old default.
		register(faces, as)
	}
}

// DefaultFont returns the family SetDefaultFont set.
func DefaultFont() string {
	mu.Lock()
	defer mu.Unlock()
	return string(defaultFamily)
}

// Resolve maps f, as text styles ask for it, to a face registered with
// RegisterFont under that family: the one with the same style and the
// closest weight (the heavier on a tie). No family means the default
// (SetDefaultFont). A family registered no other way is returned as
// is, for the shaper to match.
func Resolve(f font.Font) font.Font {
	mu.Lock()
	defer mu.Unlock()
	if f.Typeface == "" {
		f.Typeface = defaultFamily
	}
	best, found := alias{}, false
	score := func(a alias) int {
		d := int(a.weight) - int(f.Weight)
		if d < 0 {
			d = -d*2 + 1 // a lighter face loses a tie
		} else {
			d *= 2
		}
		if a.style != f.Style {
			d += 10000
		}
		return d
	}
	for _, a := range aliases {
		if a.family != f.Typeface {
			continue
		}
		if !found || score(a) < score(best) {
			best, found = a, true
		}
	}
	if !found {
		return f
	}
	return best.font
}
