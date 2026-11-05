// Package fonts holds the text shaper shared by all Giode elements.
//
// Register a font collection once before running the app, or call
// UseGofont to embed the Go fonts. When no fonts are registered the
// shaper still falls back to system fonts where the platform supports
// them.
package fonts

import (
	"fmt"

	"gioui.org/font"
	"gioui.org/font/gofont"
	"gioui.org/font/opentype"
	"gioui.org/text"

	"github.com/xmhrnx/giode/styles/properties"
)

var (
	shaper     *text.Shaper
	collection []font.FontFace
)

// Register replaces the font collection used to shape text.
func Register(faces []font.FontFace) {
	collection = faces
	shaper = text.NewShaper(text.WithCollection(faces))
}

// Collection returns the registered font collection, in the order the
// shaper loaded it.
func Collection() []font.FontFace {
	return collection
}

// UseGofont embeds the Go fonts for portable text rendering.
func UseGofont() {
	Register(gofont.Collection())
}

// Shaper returns the text shaper, or nil if no fonts were registered.
func Shaper() *text.Shaper {
	return shaper
}

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
// any registered before (with UseGofont, say, which also gives text in
// families you don't register something to fall back to):
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
// registered weight and style. Each file's family, weight and style
// replace what the file itself says, so they always match what Styles
// ask for. A collection (TTC) registers all its faces.
func RegisterFont(files ...File) error {
	faces := append([]font.FontFace(nil), collection...)
	for _, f := range files {
		parsed, err := opentype.ParseCollection(f.Data)
		if err != nil {
			return fmt.Errorf("giode: font %q: %w", f.Family, err)
		}
		for _, face := range parsed {
			face.Font.Typeface = font.Typeface(f.Family)
			face.Font.Weight = f.Weight.Weight()
			face.Font.Style = font.Regular
			if f.Italic {
				face.Font.Style = font.Italic
			}
			faces = append(faces, face)
		}
	}
	Register(faces)
	return nil
}
