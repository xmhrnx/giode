package textrender

import (
	"fmt"
	"image"
	"image/color"
	"math/rand"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

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
		if l.Dims.Size.X != dims.Size.X || l.Dims.Size.Y > dims.Size.Y || l.Dims.Size.Y < dims.Size.Y-4*len(l.Segments) {
			t.Errorf("width %d: dims %+v, want %+v", w, l.Dims, dims)
		}
		if len(l.Segments) != len(want) {
			t.Errorf("width %d: %d segments, want %d", w, len(l.Segments), len(want))
			continue
		}
		for i, s := range l.Segments {
			// A wrapped line's trailing space doesn't count toward its
			// width, and its height is that of its own glyphs - not of
			// styledtext's truncator, which may come from another face
			// and so be a few pixels taller, depending on the system's
			// fonts.
			if dx, dy := want[i].Max.X-s.Size.X, want[i].Max.Y-s.Size.Y; dx < 0 || dx > 6 || dy < 0 || dy > 3 {
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

// The layout that shapes each span once places every segment where the
// line-by-line layout did.
func TestLayoutRich_MatchesLineByLine(t *testing.T) {
	fonts.UseGofont()
	sh := fonts.Shaper()
	long := strings.Repeat("Streaming tokens arrive one at a time and the paragraph grows. ", 12)
	cases := map[string][]Span{
		"one long span": {{Size: 15, Content: long}},
		"mixed": {
			{Size: 15, Content: "Run "},
			{Size: 14, Font: font.Font{Typeface: "Go Mono"}, Content: "go test ./..."},
			{Size: 15, Content: " and read the " + strings.Repeat("documentation carefully ", 9)},
			{Size: 15, Font: font.Font{Weight: font.Bold}, Content: "docs\nnext line"},
			{Size: 15, Content: " then " + long},
		},
		"newlines":      {{Size: 15, Content: "first\n\nthird line " + long + "\nlast\n"}},
		"trailing nl":   {{Size: 15, Content: "a line\n"}, {Size: 15, Content: "next " + long}},
		"long word":     {{Size: 15, Content: "a " + strings.Repeat("x", 120) + " b " + long}},
		"bigger spans":  {{Size: 15, Content: long[:200]}, {Size: 22, Content: long[:300]}, {Size: 12, Content: long}},
		"spaces":        {{Size: 15, Content: "  lead and trail   " + long + "   "}},
		"rtl and mixed": {{Size: 15, Content: "English then עברית עברית עברית and more English " + long}},
	}
	for name, spans := range cases {
		for _, w := range []int{60, 120, 300, 700} {
			for _, p := range []RichParams{{}, {Alignment: text.Middle}, {LineHeight: 20, LineHeightScale: 1}} {
				var ops op.Ops
				gtx := layout.Context{Ops: &ops, Constraints: layout.Constraints{Max: image.Pt(w, 1<<20)}, Metric: unit.Metric{PxPerDp: 1.5, PxPerSp: 1.5}}
				got, want := layoutRich(sh, gtx, p, spans), layoutRichRef(sh, gtx, p, spans)
				if degenerate(want) {
					// The line-by-line layout put the rest of a span whose
					// word didn't fit in one block of several lines.
					continue
				}
				if len(got.Segments) != len(want.Segments) {
					t.Errorf("%s, width %d, %+v: %d segments, want %d", name, w, p, len(got.Segments), len(want.Segments))
					continue
				}
				if dy := want.Dims.Size.Y - got.Dims.Size.Y; got.Dims.Size.X != want.Dims.Size.X || dy < 0 || dy > 4*len(got.Segments) {
					t.Errorf("%s, width %d, %+v: dims %v, want %v", name, w, p, got.Dims.Size, want.Dims.Size)
				}
				for i := range got.Segments {
					g, s := got.Segments[i], want.Segments[i]
					// As in TestLayoutRich_MatchesStyledtext: a wrapped
					// line's trailing spaces, centered with the line or
					// not, and the truncator's height, which the line-by-
					// line layout counted: a few pixels per line, more
					// where the system's fonts give the truncator (and so
					// the line) a taller face, as on macOS and Windows. A
					// line starting with letters from another face may
					// sit a pixel lower or higher.
					dx, dy := s.Size.X-g.Size.X, s.Size.Y-g.Size.Y
					ox := s.Offset.X - g.Offset.X
					if p.Alignment == text.Middle || name == "rtl and mixed" || name == "spaces" {
						// Centered on a narrower line; a right-to-left run
						// starts at a fraction of a pixel.
						ox = 0
					}
					same := g.Span == s.Span && ox == 0 && s.Offset.Y-g.Offset.Y >= 0 && s.Offset.Y-g.Offset.Y <= 4*i &&
						dx >= -1 && (dx <= 7 || name == "spaces") && dy >= 0 && dy <= 3 && g.Ascent-s.Ascent >= -1 && g.Ascent-s.Ascent <= 1 &&
						g.RuneStart == s.RuneStart && g.RuneEnd == s.RuneEnd && len(g.Glyphs) == len(s.Glyphs)
					// Bidi levels are resolved over the whole paragraph now,
					// not line by line: a space between runs of opposite
					// direction may sit on the other side of a run.
					for j := 0; same && name != "rtl and mixed" && j < len(g.Glyphs); j++ {
						d := g.Glyphs[j].X - s.Glyphs[j].X
						same = d >= -64 && d <= 64 && g.Glyphs[j].Y == s.Glyphs[j].Y && g.Glyphs[j].ID == s.Glyphs[j].ID
					}
					if !same {
						t.Errorf("%s, width %d, %+v, segment %d:\n got  %+v\n want %+v", name, w, p, i, brief(g), brief(s))
						break
					}
				}
			}
		}
	}
}

// degenerate reports a segment of several lines.
func degenerate(l *RichLaid) bool {
	for _, s := range l.Segments {
		if s.Size.Y > 2*s.Ascent+s.Ascent/2 {
			return true
		}
	}
	return false
}

func brief(s Segment) string {
	return fmt.Sprintf("span %d off %v size %v ascent %d runes %d-%d glyphs %d", s.Span, s.Offset, s.Size, s.Ascent, s.RuneStart, s.RuneEnd, len(s.Glyphs))
}

// A span growing a little every frame, laid out reusing its earlier
// lines, comes out as it does laid out afresh.
func TestLayoutRich_StreamingReuse(t *testing.T) {
	fonts.UseGofont()
	sh := fonts.Shaper()
	words := strings.Fields("Streaming tokens arrive one at a time and the paragraph grows, sometimes with longer words like internationalization or short ones, a b c, and punctuation!")
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: layout.Constraints{Max: image.Pt(420, 1<<20)}, Metric: unit.Metric{PxPerDp: 1.5, PxPerSp: 1.5}}
	content := "Intro: "
	reused := false
	for i := 0; i < 400; i++ {
		content += words[i%len(words)] + " "
		if i%37 == 0 {
			content += "\n"
		}
		spans := []Span{{Size: 15, Content: "Bold start "}, {Size: 15, Content: content}}
		got := layoutRich(sh, gtx, RichParams{}, spans)
		streams.Lock()
		reused = reused || len(streams.m) > 0
		saved := streams.m
		streams.m = nil
		streams.Unlock()
		want := layoutRich(sh, gtx, RichParams{}, spans)
		streams.Lock()
		streams.m = saved
		streams.Unlock()
		if got.Dims != want.Dims || len(got.Segments) != len(want.Segments) {
			for j := range min(len(got.Segments), len(want.Segments)) {
				if brief(got.Segments[j]) != brief(want.Segments[j]) {
					t.Logf("segment %d:\n got  %s\n want %s", j, brief(got.Segments[j]), brief(want.Segments[j]))
				}
			}
			t.Fatalf("step %d: dims %v, %d segments; want %v, %d", i, got.Dims.Size, len(got.Segments), want.Dims.Size, len(want.Segments))
		}
		for j := range got.Segments {
			g, w := got.Segments[j], want.Segments[j]
			if brief(g) != brief(w) {
				t.Fatalf("step %d segment %d:\n got  %s\n want %s", i, j, brief(g), brief(w))
			}
			for k := range g.Glyphs {
				// Whether a glyph starts a paragraph depends on where the
				// shape began; nothing laying out rich text reads it.
				a, b := g.Glyphs[k], w.Glyphs[k]
				a.Flags &^= text.FlagParagraphStart
				b.Flags &^= text.FlagParagraphStart
				if a.Bounds.Empty() && b.Bounds.Empty() {
					// A space ending a wrapped line has no advance;
					// the same space shaped last on its own has one.
					a.Advance, b.Advance = 0, 0
				}
				if a != b {
					t.Fatalf("step %d segment %d glyph %d differs:\n got  %+v\n want %+v", i, j, k, g.Glyphs[k], w.Glyphs[k])
				}
			}
		}
	}
	if !reused {
		t.Error("the long span's lines were never kept")
	}
}

// A window that stops drawing lets go of what its last frame didn't
// show, and of Gio's shaper cache; what it shows stays.
func TestIdleSweep(t *testing.T) {
	fonts.UseGofont()
	defer idleSweep.Store(idleSweep.Load())
	idleSweep.Store(int64(20 * time.Millisecond))
	sh := fonts.Shaper()
	cs := layout.Constraints{Max: image.Pt(400, 1000)}
	frameOf := func(texts ...string) {
		Tick()
		for _, s := range texts {
			Layout(fonts.Shaper(), params(16), cs, s)
		}
		FrameDone()
	}
	frameOf("gone soon", "still here")
	frameOf("still here")
	if _, _, plain, _ := CacheStats(); plain < 2 {
		t.Fatalf("%d layouts cached, want both", plain)
	}
	deadline := time.Now().Add(2 * time.Second)
	for fonts.Shaper() == sh && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if fonts.Shaper() == sh {
		t.Fatal("the idle window kept its shaper and Gio's cache")
	}
	layouts.Lock()
	var kept []string
	for k := range layouts.m {
		kept = append(kept, k.str)
	}
	layouts.Unlock()
	if len(kept) != 1 || kept[0] != "still here" {
		t.Errorf("after the idle sweep the layouts are %q, want only what the last frame drew", kept)
	}
}

// The shaper is replaced once enough text has been shaped with it.
func TestShaperRefresh(t *testing.T) {
	fonts.UseGofont()
	sh := fonts.Shaper()
	cs := layout.Constraints{Max: image.Pt(400, 1<<20)}
	para := strings.Repeat("Text that keeps arriving, a new paragraph every frame. ", 40)
	for i := 0; fonts.Shaper() == sh && i < 400; i++ {
		Tick()
		Layout(fonts.Shaper(), params(16), cs, fmt.Sprint(i, para))
		FrameDone()
	}
	if fonts.Shaper() == sh {
		t.Error("the shaper was never replaced")
	}
}
