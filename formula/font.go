package formula

import (
	"sync"

	"github.com/go-fonts/stix/stix2mathregular"

	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/internal/mathhook"
	"github.com/xmhrnx/giode/styles"
)

// Family is the font formulas are set in, STIX Two Math (SIL Open Font
// License), registered when the package is imported.
const Family = "STIX Two Math"

var registerOnce sync.Once

func init() {
	Register()
	// Markdown sets $...$ and $$...$$ with MarkdownOptions.Math.
	mathhook.Set(func(src string, display bool, st styles.Styles) (mathhook.Element, error) {
		if err := Check(src); err != nil {
			return nil, err
		}
		if display {
			return New(src, st), nil
		}
		return Inline(src, st), nil
	})
}

// Register adds STIX Two Math to the fonts text is shaped with. Importing
// the package does it; call it again after replacing the fonts with
// giode.RegisterFonts (RegisterFont adds to them and needs nothing). Its
// symbols then also stand in for those other fonts lack, in all text:
// ∇, ℏ, ⇒, ∝, ∮ and the rest.
func Register() {
	registerOnce.Do(register)
	if !registered() {
		register()
	}
}

func register() {
	fonts.RegisterFont(fonts.File{Family: Family, Data: stix2mathregular.TTF})
}

// registered reports whether the font is in the current collection.
func registered() bool {
	for _, f := range fonts.Collection() {
		if string(f.Font.Typeface) == Family {
			return true
		}
	}
	return false
}
