package formula

import (
	"image"
	"strings"
	"testing"

	giofont "gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"golang.org/x/image/math/fixed"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

func size(t *testing.T, src string, inline bool) layout.Dimensions {
	t.Helper()
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Constraints{Max: image.Pt(2000, 2000)}}
	f := New(src, styles.Styles{FontSize: 20})
	if inline {
		f = Inline(src, styles.Styles{FontSize: 20})
	}
	return f.Layout(gtx)
}

func TestLayout(t *testing.T) {
	x := size(t, "x", false)
	if x.Size.X <= 0 || x.Size.Y <= 0 || x.Size.Y > 30 {
		t.Fatalf("x: %v", x.Size)
	}
	// A fraction stands taller than a letter, above and below the baseline.
	f := size(t, `\frac{a}{b}`, false)
	if f.Size.Y < 2*x.Size.Y || f.Baseline < x.Baseline+5 {
		t.Errorf(`\frac{a}{b}: %v baseline %d, x %v baseline %d`, f.Size, f.Baseline, x.Size, x.Baseline)
	}
	// Inline, a sum's limits go beside it, and its fraction is smaller.
	d := size(t, `\sum_{n=1}^{\infty}\frac{1}{n^2}`, false)
	i := size(t, `\sum_{n=1}^{\infty}\frac{1}{n^2}`, true)
	if i.Size.Y >= d.Size.Y {
		t.Errorf("inline sum %v isn't shorter than display %v", i.Size, d.Size)
	}
	// Aligned lines stack.
	one := size(t, `E = mc^2`, false)
	two := size(t, `E &= mc^2 \\ &= \gamma m_0 c^2`, false)
	if two.Size.Y < 2*one.Size.Y-4 {
		t.Errorf("two lines %v, one %v", two.Size, one.Size)
	}
	// Twice the font size, twice the formula.
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Constraints{Max: image.Pt(4000, 4000)}}
	big := New(`\sqrt{\frac{2GM}{r}}`, styles.Styles{FontSize: 40}).Layout(gtx)
	small := size(t, `\sqrt{\frac{2GM}{r}}`, false)
	if r := float32(big.Size.X) / float32(small.Size.X); r < 1.8 || r > 2.2 {
		t.Errorf("at 40 %v, at 20 %v: not twice", big.Size, small.Size)
	}
}

// Malformed formulas lay out without panicking.
func TestMalformed(t *testing.T) {
	for _, src := range []string{
		"", "{", "}", "^", "_", "x^", "x_", "^^^", `\`, `\frac`, `\frac{a}`, `\sqrt[`, `\sqrt[3`,
		`\left(`, `\right)`, `\left( x`, `\left\{ x \right`, `\begin{aligned}`, `&&&`, `\\\\\\`,
		`\text{`, `\vec`, `\big`, `x''''`, `\not`, `\unknown{x}`, `\mathrm{`, strings.Repeat("{", 50),
		strings.Repeat(`\frac{1}{`, 20),
		`\begin{pmatrix}`, `\begin{pmatrix} a & b \\`, `\end{pmatrix}`, `\begin{cases} x & \text{if`,
		`\begin{unknownenv} x \end{unknownenv}`, `\color{red}`, `{\color{red} x} y`, `\textcolor{nope}{x}`,
		`\overset{a}`, `\stackrel`, `\boxed`, `\phantom`, `\mathcal`, `\mathbb{}`, `\begin{matrix}\end{matrix}`,
	} {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%q panicked: %v", src, r)
				}
			}()
			size(t, src, false)
		}()
	}
}

func TestParse(t *testing.T) {
	ns := parse(`a_1^2 + \alpha \leq \frac{x}{y}`)
	if len(ns) != 5 {
		t.Fatalf("%d nodes: %#v", len(ns), ns)
	}
	s, ok := ns[0].(scripts)
	if !ok || len(s.sub) != 1 || len(s.sup) != 1 {
		t.Errorf("a_1^2 parsed as %#v", ns[0])
	}
	if a, ok := s.base.(atom); !ok || a.s != "𝑎" {
		t.Errorf("a isn't math italic: %#v", s.base)
	}
	if a := ns[1].(atom); a.cls != clsBin {
		t.Errorf("+ is %v", a.cls)
	}
	if a := ns[2].(atom); a.s != "𝛼" {
		t.Errorf(`\alpha is %q, want the italic α`, a.s)
	}
	if a := ns[3].(atom); a.s != "≤" || a.cls != clsRel {
		t.Errorf(`\leq is %#v`, a)
	}
	if _, ok := ns[4].(frac); !ok {
		t.Errorf(`\frac is %#v`, ns[4])
	}
	if r := rows(parse(`a &= b \\ &= c`)); len(r) != 2 || len(r[0]) != 2 || len(r[1]) != 2 {
		t.Errorf("rows %d %v", len(r), r)
	}
	if a := parse(`\mathrm{d}x`)[0].(group).body[0].(atom); a.s != "d" {
		t.Errorf(`\mathrm{d} is %q`, a.s)
	}
}

// Importing the package gives plain text the math font's symbols.
func TestPlainTextGetsSymbols(t *testing.T) {
	sh := fonts.Shaper()
	for _, r := range "∇ℏ⇒∝∮" {
		sh.LayoutString(text.Parameters{Font: fonts.Resolve(giofont.Font{}), PxPerEm: fixed.I(20), MaxWidth: 1000}, string(r))
		g, ok := sh.NextGlyph()
		if !ok || g.Bounds.Empty() {
			t.Errorf("%c has no glyph in plain text", r)
		}
		for ok {
			_, ok = sh.NextGlyph()
		}
	}
}

// In a row aligned on baselines, a formula sits on the text's baseline.
func TestBaselineInRow(t *testing.T) {
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Constraints{Max: image.Pt(2000, 2000)}}
	elements.BeginSemantics()
	elements.Row(styles.Styles{Align: properties.AlignBaseline},
		elements.Text("Speed", styles.Styles{FontSize: 20}),
		New(`\frac{d}{t}`, styles.Styles{FontSize: 20}),
	).Layout(gtx)
	var textB, fracB image.Rectangle
	for _, n := range elements.EndSemantics() {
		switch n.Label {
		case "Speed":
			textB = n.Bounds
		case `\frac{d}{t}`:
			fracB = n.Bounds
		}
	}
	// The text's baseline is about 80% down its box; the fraction's
	// bar (its axis) a little above that.
	textBase := textB.Min.Y + textB.Dy()*4/5
	if fracB.Min.Y >= textB.Min.Y || fracB.Max.Y <= textB.Max.Y {
		t.Errorf("fraction %v doesn't extend above and below the text %v", fracB, textB)
	}
	if mid := (fracB.Min.Y + fracB.Max.Y) / 2; mid > textBase || mid < textB.Min.Y {
		t.Errorf("fraction centered at %d, text %v (baseline ~%d)", mid, textB, textBase)
	}
}

func TestAlphabetsAndEnvironments(t *testing.T) {
	text := func(ns []node) string {
		var b strings.Builder
		var walk func([]node)
		walk = func(ns []node) {
			for _, n := range ns {
				switch n := n.(type) {
				case atom:
					b.WriteString(n.s)
				case group:
					walk(n.body)
				case scripts:
					walk([]node{n.base})
					walk(n.sup)
					walk(n.sub)
				}
			}
		}
		walk(ns)
		return b.String()
	}
	for src, want := range map[string]string{
		`\mathcal{E}`:                "ℰ",
		`\mathcal{A}`:                "𝒜",
		`\mathscr{L}`:                "ℒ",
		`\mathbb{R}`:                 "ℝ",
		`\mathbb{Z}^3`:               "ℤ3",
		`\mathfrak{g}`:               "𝔤",
		`\mathsf{A}`:                 "𝖠",
		`\mathtt{x}`:                 "𝚡",
		`\boldsymbol{\omega}`:        "𝝎",
		`\boldsymbol{r}`:             "𝒓",
		`M_\odot`:                    "𝑀⊙",
		`M_{\astrosun}`:              "𝑀☉",
		`☉ ℰ`:                        "☉ℰ",
		`\therefore \Longrightarrow`: "∴⟹",
	} {
		if got := text(parse(src)); got != want {
			t.Errorf("%s: %q, want %q", src, got, want)
		}
	}
	ns := parse(`\begin{pmatrix} a & b \\ c & d \end{pmatrix} x`)
	if len(ns) != 2 {
		t.Fatalf("pmatrix and x: %d nodes %#v", len(ns), ns)
	}
	e, ok := ns[0].(env)
	if !ok || e.name != "pmatrix" || len(rows(e.body)) != 2 || len(rows(e.body)[0]) != 2 {
		t.Errorf("pmatrix parsed as %#v", ns[0])
	}
	if st, ok := parse(`\stackrel{\text{def}}{=}`)[0].(stack); !ok || st.cls != clsRel || st.over == nil {
		t.Errorf("stackrel: %#v", parse(`\stackrel{\text{def}}{=}`)[0])
	}
	// \color colors the rest of its group only.
	ns = parse(`{a \color{red} b c} d`)
	g := ns[0].(group)
	if c, ok := g.body[1].(colored); !ok || len(c.body) != 2 {
		t.Errorf("color in a group: %#v", g.body)
	}
	if a, ok := ns[1].(atom); !ok || a.s != "𝑑" {
		t.Errorf("after the group: %#v", ns[1:])
	}
	// The environments lay out.
	for _, src := range []string{
		`R = \begin{pmatrix} \cos\theta & -\sin\theta \\ \sin\theta & \cos\theta \end{pmatrix}`,
		`|x| = \begin{cases} x & x \geq 0 \\ -x & x < 0 \end{cases}`,
		`\det\begin{vmatrix} a & b \\ c & d \end{vmatrix}`,
		`\begin{aligned} a &= b \\ &= c \end{aligned}`,
	} {
		if d := size(t, src, false); d.Size.Y < 40 {
			t.Errorf("%s: %v, want two rows tall", src, d.Size)
		}
	}
}
