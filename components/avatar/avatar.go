// Package avatar provides avatars: a person's or assistant's picture,
// or their initials on a color derived from their name, in a circle.
package avatar

import (
	"hash/fnv"
	"image"
	"strings"
	"unicode"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/components/imageutil"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// palette holds the backgrounds initials are drawn on; a name always
// picks the same one.
var palette = []properties.Color{"#ef4444", "#f97316", "#eab308", "#22c55e", "#14b8a6", "#3b82f6", "#6366f1", "#a855f7", "#ec4899", "#64748b"}

// Avatar is a circle showing an image, or the initials of a name. It's
// stateless: build it in the view function every frame.
type Avatar struct {
	name string
	img  image.Image
	st   styles.Styles
}

// New returns an avatar for name. The styles argument is optional:
// Width sets the diameter (default 40dp), Background overrides the
// derived color, Color the initials (default white).
func New(name string, st ...styles.Styles) *Avatar {
	a := &Avatar{name: name}
	if len(st) > 0 {
		a.st = st[0]
	}
	return a
}

// Image shows img, cropped to the circle, instead of the initials. It
// returns a for chaining.
func (a *Avatar) Image(img image.Image) *Avatar {
	a.img = img
	return a
}

// Initials returns up to two letters for name: the first letters of
// its first and last words.
func Initials(name string) string {
	words := strings.FieldsFunc(name, func(r rune) bool { return unicode.IsSpace(r) || r == '-' || r == '_' || r == '.' })
	var out []rune
	for i, w := range words {
		if i != 0 && i != len(words)-1 {
			continue
		}
		for _, r := range w {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				out = append(out, unicode.ToUpper(r))
				break
			}
		}
	}
	return string(out)
}

// Color returns the background derived from name.
func Color(name string) properties.Color {
	h := fnv.New32a()
	h.Write([]byte(strings.ToLower(name)))
	return palette[h.Sum32()%uint32(len(palette))]
}

// Layout lays out the avatar.
func (a *Avatar) Layout(gtx layout.Context) layout.Dimensions {
	d := a.st.Width
	if d <= 0 {
		d = 40
	}
	px := gtx.Dp(unit.Dp(d))
	size := image.Pt(px, px)
	gtx.Constraints = layout.Exact(size)
	defer clip.Ellipse(image.Rectangle{Max: size}).Push(gtx.Ops).Pop()
	if a.img != nil {
		imageutil.New(a.img, styles.Styles{Width: d, Height: properties.Height(d), Fit: properties.FitCover}).Layout(gtx)
		return layout.Dimensions{Size: size}
	}
	bg := a.st.Background
	if bg == "" {
		bg = Color(a.name)
	}
	fg := a.st.Color
	if fg == "" {
		fg = "#ffffff"
	}
	fs := a.st.FontSize
	if fs == 0 {
		fs = properties.FontSize(float32(d) * 0.4)
	}
	elements.Box(styles.Styles{Width: d, Height: properties.Height(d), Background: bg, Justify: properties.JustifyCenter, Align: properties.AlignCenter},
		elements.Text(Initials(a.name), styles.Styles{Color: fg, FontSize: fs, FontWeight: properties.FontWeightSemiBold})).Layout(gtx)
	elements.RecordSemantic(gtx, elements.SemanticNode{Kind: elements.SemanticImage, Label: a.name}, size)
	return layout.Dimensions{Size: size}
}
