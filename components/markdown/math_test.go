package markdown

import (
	"image"
	"strings"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/fonts"
	_ "github.com/xmhrnx/giode/formula"
	gast "github.com/yuin/goldmark/ast"
)

// maths returns the formulas parsed from src: inline ones as "i:src",
// display ones as "d:src".
func maths(src string) []string {
	var out []string
	gast.Walk(parse(src, true), func(n gast.Node, entering bool) (gast.WalkStatus, error) {
		if !entering {
			return gast.WalkContinue, nil
		}
		switch n := n.(type) {
		case *mathInline:
			kind := "i:"
			if n.display {
				kind = "I:"
			}
			out = append(out, kind+n.src)
		case *mathBlock:
			kind := "d:"
			if !n.closed {
				kind = "open:"
			}
			out = append(out, kind+n.source())
		}
		return gast.WalkContinue, nil
	})
	return out
}

func TestMathParsing(t *testing.T) {
	for _, c := range []struct{ src, want string }{
		{"Euler: $e^{i\\pi}+1=0$.", "i:e^{i\\pi}+1=0"},
		{"Prices: $5 and $10 today.", ""},
		{"From $5 to $x$.", "i:x"},
		{"a $ x$ b", ""},
		{"a $x $b", ""},
		{"costs \\$5, while $y$ is free", "i:y"},
		{"inline \\(x^2\\) here", "i:x^2"},
		{"code `$x$` stays", ""},
		{"over $a +\nb$ lines", "i:a +\nb"},
		{"text $$\\sum x$$ text", "I:\\sum x"},
		{"$$\n\\int_0^1 x\\,dx\n$$", "d:\\int_0^1 x\\,dx"},
		{"$$ E = mc^2 $$", "d:E = mc^2"},
		{"\\[\na^2 + b^2\n\\]", "d:a^2 + b^2"},
		{"Then\n$$\nx\n$$\nafter", "d:x"},
		{"$$\nx = \\frac{1}{2", "open:x = \\frac{1}{2"},
		{"5$ and 6$", ""},
	} {
		if got := strings.Join(maths(c.src), "|"); got != c.want {
			t.Errorf("%q: %q, want %q", c.src, got, c.want)
		}
	}
	// Without Math, dollars are text.
	if n := parse("$x$", false); n.FirstChild().FirstChild().Kind() == kindMathInline {
		t.Error("math without Math")
	}
}

func layoutMath(t *testing.T, src string) (layout.Dimensions, []elements.SemanticNode) {
	t.Helper()
	fonts.UseGofont()
	elements.SetWindowSize(image.Pt(500, 2000))
	t.Cleanup(func() { elements.SetWindowSize(image.Point{}) })
	elements.BeginFrame()
	elements.BeginSemantics()
	var ops op.Ops
	dims := New(src, Options{Math: true}).Layout(layout.Context{Ops: &ops, Constraints: layout.Constraints{Max: image.Pt(500, 1e6)}})
	return dims, elements.EndSemantics()
}

func labels(nodes []elements.SemanticNode, kind elements.SemanticKind) []string {
	var out []string
	for _, n := range nodes {
		if n.Kind == kind {
			out = append(out, n.Label)
		}
	}
	return out
}

// Formulas are set, and their paragraph reads (and copies) as their
// source; one that doesn't read shows its source.
func TestMathRendering(t *testing.T) {
	_, nodes := layoutMath(t, "Euler: $e^{i\\pi}+1=0$, and\n\n$$\n\\int_0^1 x\\,dx = \\frac12\n$$\n\nbroken $\\frac{a}{$ here")
	all := strings.Join(labels(nodes, elements.SemanticText), "\n")
	for _, want := range []string{
		"Euler: $e^{i\\pi}+1=0$, and", // the paragraph, copied with its source
		"e^{i\\pi}+1=0",               // the formula itself
		"$$\n\\int_0^1 x\\,dx = \\frac12\n$$",
		"broken $\\frac{a}{$ here",
	} {
		if !strings.Contains(all, want) {
			t.Errorf("no %q in:\n%s", want, all)
		}
	}
	if strings.Contains(all, "\n\\frac{a}{\n") {
		t.Error("the broken formula was set")
	}
}

// An inline formula taller than the text makes its line taller; one
// that fits leaves the paragraph as tall as plain text.
func TestMathLineHeight(t *testing.T) {
	plain, _ := layoutMath(t, "x plus y")
	small, _ := layoutMath(t, "$x$ plus $y$")
	tall, _ := layoutMath(t, "$\\dfrac{\\sum_{i=1}^n a_i}{\\prod_{j=1}^m b_j}$ plus $y$")
	if d := small.Size.Y - plain.Size.Y; d < 0 || d > 3 {
		t.Errorf("small formulas: height %d, plain text %d", small.Size.Y, plain.Size.Y)
	}
	if tall.Size.Y < plain.Size.Y+20 {
		t.Errorf("a tall formula: height %d, plain text %d", tall.Size.Y, plain.Size.Y)
	}
}
