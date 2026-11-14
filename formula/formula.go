// Package formula typesets mathematics written in LaTeX notation:
// fractions, roots, superscripts and subscripts, sums and integrals with
// their limits, delimiters sized to what they enclose, accents (vectors,
// hats, dots), Greek, and lines aligned at & for derivations.
//
//	formula.New(`v = \sqrt{\frac{2GM}{r}}`, giode.Styles{FontSize: 20})
//	formula.New(`\oint \vec{E}\cdot d\vec{A} = \frac{Q}{\varepsilon_0}`)
//	formula.New(`F &= ma \\ &= m\frac{dv}{dt}`) // aligned lines
//
// Formulas are set in STIX Two Math, which importing the package adds
// to the app's fonts (about 0.8 MB); its symbols - ∇, ℏ, ⇒, ∝ and the
// rest - then also show in ordinary Text.
package formula

import (
	"image"
	"math"
	"reflect"
	"sync"

	"gioui.org/layout"
	"gioui.org/text"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Formula is an element showing a formula.
type Formula struct {
	src    string
	st     styles.Styles
	inline bool
}

// New returns a formula in display style, as a formula on a line of its
// own is set: large operators and fractions at full size. The styles
// argument is optional: FontSize (the formula's size), Color, and the
// box styles (Padding, Margin, Background, Width, alignment).
//
// It understands \frac (\dfrac, \tfrac), \sqrt and \sqrt[n], ^ and _,
// \sum \prod \int \iint \oint (with \limits and \nolimits), \lim \max
// \min and the named functions (\sin, \log...), \left( ... \right) with
// ( [ \{ | \| \langle and . delimiters, \big and \Big, accents (\vec
// \hat \bar \overline \dot \ddot \tilde \underline), \text, \mathrm,
// \mathbf, \operatorname, Greek letters, the usual relations, arrows
// and operators (\nabla \partial \hbar \propto \Rightarrow \approx
// \leq \pm \times \cdot \infty ...), spaces (\, \; \quad), and \\ and &
// for aligned lines. Letters are set in italic, as variables; write
// \mathrm{d}x for an upright d.
func New(src string, st ...styles.Styles) *Formula {
	f := &Formula{src: src}
	if len(st) > 0 {
		f.st = st[0]
	}
	return f
}

// Inline returns a formula in text style, as one within a line of text
// is set: smaller fractions, operators and limits.
func Inline(src string, st ...styles.Styles) *Formula {
	f := New(src, st...)
	f.inline = true
	return f
}

// Source returns the formula as written.
func (f *Formula) Source() string { return f.src }

// Layout lays the formula out at its natural size; its baseline is the
// first line's.
func (f *Formula) Layout(gtx layout.Context) layout.Dimensions {
	if bs := f.boxStyles(); !reflect.ValueOf(bs).IsZero() {
		return elements.Box(bs, elements.Raw(f.layoutSelf)).Layout(gtx)
	}
	// Without box styles the formula keeps its baseline, for lining it up
	// with text in a row.
	return f.layoutSelf(gtx)
}

// boxStyles are the styles of the box around the formula.
func (f *Formula) boxStyles() styles.Styles {
	st := f.st
	st.FontSize, st.Color = 0, ""
	return st
}

func (f *Formula) layoutSelf(gtx layout.Context) layout.Dimensions {
	px := float32(gtx.Sp(properties.ResolveFontSize(f.st.FontSize)))
	b := laidOut(f.src, px, f.inline)
	asc, desc := float32(math.Ceil(float64(b.asc))), float32(math.Ceil(float64(b.desc)))
	size := image.Pt(int(math.Ceil(float64(b.w))), int(asc+desc))
	if b.draw != nil {
		d := &drawer{ops: gtx.Ops, col: elements.ResolveColor(string(f.st.Color)), shaper: b.shaper}
		b.draw(d, 0, asc)
	}
	elements.RecordSemantic(gtx, elements.SemanticNode{Kind: elements.SemanticText, Label: f.src}, size)
	return layout.Dimensions{Size: size, Baseline: int(desc)}
}

// laid is a cached layout and the shaper it was made with.
type laid struct {
	box
	shaper *text.Shaper
}

type cacheKey struct {
	src    string
	px     float32
	inline bool
}

var (
	cacheMu sync.Mutex
	cache   = map[cacheKey]laid{}
)

// laidOut returns the formula's layout, from the cache when the fonts
// haven't changed since.
func laidOut(src string, px float32, inline bool) laid {
	l := newLayouter()
	key := cacheKey{src, px, inline}
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if c, ok := cache[key]; ok && c.shaper == l.shaper {
		return c
	}
	if len(cache) > 512 {
		clear(cache)
	}
	st := mstyle{level: 0, base: px}
	if inline {
		st.level = 1
	}
	c := laid{box: l.layoutFormula(parse(src), st), shaper: l.shaper}
	cache[key] = c
	return c
}
