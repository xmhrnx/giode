package textrender

import (
	"image"
	"image/color"
	"math/rand"
	"os"
	"runtime"
	"strings"
	"testing"

	"gioui.org/font"
	"gioui.org/gpu/headless"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/x/styledtext"
	"golang.org/x/image/math/fixed"

	"github.com/xmhrnx/giode/fonts"
)

func params(size int) text.Parameters {
	return text.Parameters{PxPerEm: fixed.I(size), Font: font.Font{}}
}

func TestGlyphIDLayout(t *testing.T) {
	fonts.UseGofont()
	sh := fonts.Shaper()
	fs := faces(sh)
	if len(fs) == 0 {
		t.Fatal("no faces")
	}
	for _, f := range []font.Font{{}, {Weight: font.Bold}, {Style: font.Italic}, {Typeface: "Go Mono"}} {
		p := params(17)
		p.Font = f
		p.MaxWidth = 1000
		sh.LayoutString(p, "Hello")
		for g, ok := sh.NextGlyph(); ok; g, ok = sh.NextGlyph() {
			if g.Flags&text.FlagParagraphStart != 0 && g.Runes == 0 {
				continue
			}
			ppem, faceIdx, gid := splitGlyphID(g.ID)
			if ppem != fixed.I(17) || faceIdx >= len(fs) || gid == 0 {
				t.Fatalf("%v: glyph %x splits to ppem %v face %d gid %d", f, g.ID, ppem, faceIdx, gid)
			}
		}
	}
}

var samples = []string{
	"Hello, world",
	"The quick brown fox jumps over the lazy dog. " + strings.Repeat("Wrapping text keeps going. ", 8),
	"Line one\nLine two\n\nAfter a blank line\n",
	"",
	"tabs\tand  spaces   ",
}

// labelGlyphs returns what widget.Label lays out, through Gio.
func labelGlyphs(sh *text.Shaper, p text.Parameters, cs layout.Constraints, str string) (layout.Dimensions, []text.Glyph) {
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: cs, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
	lb := widget.Label{MaxLines: p.MaxLines, Truncator: p.Truncator, Alignment: p.Alignment, LineHeight: unit.Sp(float32(p.LineHeight) / 64), LineHeightScale: p.LineHeightScale}
	dims := lb.Layout(gtx, sh, p.Font, unit.Sp(float32(p.PxPerEm)/64), str, op.CallOp{})
	p.MaxWidth, p.MinWidth = cs.Max.X, cs.Min.X
	sh.LayoutString(p, str)
	var gs []text.Glyph
	for g, ok := sh.NextGlyph(); ok; g, ok = sh.NextGlyph() {
		gs = append(gs, g)
	}
	return dims, gs
}

func TestLayout_MatchesLabel(t *testing.T) {
	fonts.UseGofont()
	sh := fonts.Shaper()
	for _, str := range samples {
		for _, p := range []text.Parameters{params(16), {PxPerEm: fixed.I(14), MaxLines: 2, Truncator: "…"}, {PxPerEm: fixed.I(15), LineHeight: fixed.I(24), LineHeightScale: 1}, {PxPerEm: fixed.I(16), Alignment: text.Middle}} {
			cs := layout.Constraints{Max: image.Pt(300, 1000)}
			want, gs := labelGlyphs(sh, p, cs, str)
			l := layoutWhole(sh, withWidth(p, cs), cs, str)
			if l.Dims != want {
				t.Errorf("%q %+v: dims %+v, want %+v", str, p, l.Dims, want)
			}
			for i, g := range l.Glyphs {
				if i >= len(gs) || g != gs[i] {
					t.Fatalf("%q: glyph %d differs", str, i)
				}
			}
		}
	}
}

func withWidth(p text.Parameters, cs layout.Constraints) text.Parameters {
	p.MaxWidth, p.MinWidth = cs.Max.X, cs.Min.X
	return p
}

func TestLayoutParagraphs_MatchesWhole(t *testing.T) {
	fonts.UseGofont()
	sh := fonts.Shaper()
	r := rand.New(rand.NewSource(1))
	words := []string{"alpha", "beta", "gamma", "delta", "epsilon", "**bold**", "x", "longerwordhere", "1234567890"}
	for n := 0; n < 30; n++ {
		var b strings.Builder
		for b.Len() < 300+r.Intn(900) {
			b.WriteString(words[r.Intn(len(words))])
			switch r.Intn(12) {
			case 0:
				b.WriteString("\n")
			case 1:
				b.WriteString("\n\n")
			default:
				b.WriteString(" ")
			}
		}
		if r.Intn(2) == 0 {
			b.WriteString("\n")
		}
		str := b.String()
		for _, p := range []text.Parameters{params(16), {PxPerEm: fixed.I(15), LineHeight: fixed.I(22), LineHeightScale: 1}, {PxPerEm: fixed.I(13), LineHeightScale: 1.5}} {
			cs := layout.Constraints{Max: image.Pt(280+r.Intn(300), 1<<20)}
			whole := layoutWhole(sh, withWidth(p, cs), cs, str)
			if !splittable(p, str) {
				t.Fatalf("sample not splittable")
			}
			split := layoutParagraphs(sh, withWidth(p, cs), cs, str)
			if split.Dims != whole.Dims {
				t.Fatalf("sample %d %+v: dims %+v, want %+v", n, p, split.Dims, whole.Dims)
			}
			var gs []text.Glyph
			lines := 0
			split.EachLine(func(line []text.Glyph, dy int) {
				lines++
				for _, g := range line {
					g.Y += int32(dy)
					gs = append(gs, g)
				}
			})
			if len(gs) != len(whole.Glyphs) {
				t.Fatalf("sample %d: %d glyphs, want %d", n, len(gs), len(whole.Glyphs))
			}
			for i := range whole.Glyphs {
				a, b := gs[i], whole.Glyphs[i]
				if a.ID != b.ID || a.X != b.X || a.Y != b.Y {
					t.Fatalf("sample %d glyph %d: (%v,%v) want (%v,%v)", n, i, a.X, a.Y, b.X, b.Y)
				}
			}
			if lines != len(whole.Lines) {
				t.Fatalf("sample %d: %d lines, want %d", n, lines, len(whole.Lines))
			}
		}
	}
}

func TestLayout_Cached(t *testing.T) {
	fonts.UseGofont()
	sh := fonts.Shaper()
	cs := layout.Constraints{Max: image.Pt(300, 1000)}
	a := Layout(sh, params(16), cs, "cached text")
	b := Layout(sh, params(16), cs, "cached text")
	if a != b {
		t.Error("same text and parameters laid out twice")
	}
	if c := Layout(sh, params(17), cs, "cached text"); c == a {
		t.Error("different size served from cache")
	}
	// Paragraphs longer than the split threshold, each ending in a
	// newline, don't split further.
	long := strings.Repeat(strings.Repeat("word ", 80)+"\n", 3)
	l := Layout(sh, params(16), layout.Constraints{Max: image.Pt(400, 1<<20)}, long)
	if want := layoutWhole(sh, withWidth(params(16), layout.Constraints{Max: image.Pt(400, 1<<20)}), layout.Constraints{Max: image.Pt(400, 1<<20)}, long); l.Dims != want.Dims {
		t.Errorf("long paragraphs: dims %+v, want %+v", l.Dims, want.Dims)
	}
}

func TestLayoutRich_MatchesStyledtext(t *testing.T) {
	fonts.UseGofont()
	sh := fonts.Shaper()
	spans := []Span{
		{Size: 15, Content: "Run "},
		{Size: 14, Font: font.Font{Typeface: "Go Mono"}, Content: "go test ./..."},
		{Size: 15, Content: " and read the " + strings.Repeat("documentation carefully ", 5)},
		{Size: 15, Font: font.Font{Weight: font.Bold}, Content: "docs\nnext line"},
	}
	for _, w := range []int{120, 300, 900} {
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Constraints: layout.Constraints{Max: image.Pt(w, 1000)}, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
		st := make([]styledtext.SpanStyle, len(spans))
		for i, s := range spans {
			st[i] = styledtext.SpanStyle{Font: s.Font, Size: s.Size, Content: s.Content, Color: color.NRGBA{A: 255}}
		}
		var want []image.Rectangle
		dims := styledtext.Text(sh, st...).Layout(gtx, func(gtx layout.Context, i int, d layout.Dimensions) {
			want = append(want, image.Rectangle{Max: d.Size})
		})
		l := layoutRich(sh, gtx, RichParams{}, spans)
		if l.Dims != dims {
			t.Errorf("width %d: dims %+v, want %+v", w, l.Dims, dims)
		}
		if len(l.Segments) != len(want) {
			t.Errorf("width %d: %d segments, want %d", w, len(l.Segments), len(want))
			continue
		}
		for i, s := range l.Segments {
			if s.Size != want[i].Max {
				t.Errorf("width %d segment %d: size %v, want %v", w, i, s.Size, want[i].Max)
			}
		}
	}
}

func render(t testing.TB, w, h int, fn func(ops *op.Ops)) *image.RGBA {
	if runtime.GOOS == "linux" && os.Getenv("EGL_PLATFORM") == "" {
		os.Setenv("EGL_PLATFORM", "surfaceless")
	}
	win, err := headless.NewWindow(w, h)
	if err != nil {
		t.Skipf("no offscreen renderer: %v", err)
	}
	defer win.Release()
	var ops op.Ops
	paint.Fill(&ops, color.NRGBA{255, 255, 255, 255})
	fn(&ops)
	if err := win.Frame(&ops); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	win.Screenshot(img)
	return img
}

// The images match Gio's paths: same glyphs in the same places, with
// small antialiasing differences.
func TestDrawRun_MatchesVector(t *testing.T) {
	fonts.UseGofont()
	sh := fonts.Shaper()
	cs := layout.Constraints{Max: image.Pt(560, 400)}
	str := "Quick brown fox, jumps! ffi Wg 0123 @#& ßé\nSecond line of text, italic-ish"
	l := layoutWhole(sh, withWidth(params(19), cs), cs, str)
	draw := func(vec bool) *image.RGBA {
		return render(t, 600, 120, func(ops *op.Ops) {
			SetVectorText(vec)
			defer SetVectorText(false)
			defer op.Offset(image.Pt(10, 10)).Push(ops).Pop()
			l.Draw(ops, sh, color.NRGBA{A: 255})
			clearRuns()
		})
	}
	vec, img := draw(true), draw(false)
	var sum, n int
	worst := 0
	for i := 0; i < len(vec.Pix); i += 4 {
		d := int(vec.Pix[i]) - int(img.Pix[i])
		if d < 0 {
			d = -d
		}
		sum += d
		n++
		worst = max(worst, d)
	}
	if mean := float64(sum) / float64(n); mean > 2 {
		t.Errorf("mean difference %.2f, worst %d: images don't match the paths", mean, worst)
	}
	if os.Getenv("TEXT_SHOT") != "" {
		save(os.Getenv("TEXT_SHOT")+"-vector.png", vec)
		save(os.Getenv("TEXT_SHOT")+"-image.png", img)
	}
}

func clearRuns() {
	runCache.Lock()
	runCache.m = nil
	runCache.bytes = 0
	runCache.Unlock()
}
