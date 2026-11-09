package text

import (
	"image"
	"testing"

	"gioui.org/layout"
	gtext "gioui.org/text"
	"golang.org/x/image/math/fixed"

	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/internal/textrender"
	"github.com/xmhrnx/giode/styles/properties"
)

func laidOut(t *testing.T, s string, width int, align gtext.Alignment) *textrender.Laid {
	t.Helper()
	fonts.UseGofont()
	cs := layout.Constraints{Min: image.Pt(width, 0), Max: image.Pt(width, 1000)}
	return textrender.Layout(fonts.Shaper(), gtext.Parameters{PxPerEm: fixed.I(16), Alignment: align}, cs, s)
}

// A decoration covers the text, not the box it was laid out in, which
// a FlexGrow or a minimum width makes wider.
func TestDecorationCoversText(t *testing.T) {
	l := laidOut(t, "Done", 400, gtext.Start)
	lines := decorationLines(l, properties.TextDecorationLineThrough, 1)
	if len(lines) != 1 {
		t.Fatalf("%d lines, want 1: %v", len(lines), lines)
	}
	r := lines[0]
	if r.Min.X > 2 || r.Max.X < 25 || r.Max.X > 60 {
		t.Errorf("line-through %v: want it from the text's start to its end (about 40px of 400)", r)
	}
	// Through the letters: below the line's top, above its baseline
	// (which Dimensions counts from the bottom).
	baseline := l.Dims.Size.Y - l.Dims.Baseline
	if r.Min.Y < 2 || r.Max.Y > baseline-2 {
		t.Errorf("line-through at y %d-%d, baseline %d", r.Min.Y, r.Max.Y, baseline)
	}
	u := decorationLines(l, properties.TextDecorationUnderline, 1)[0]
	if u.Min.Y <= baseline || u.Max.Y > l.Dims.Size.Y {
		t.Errorf("underline at y %d-%d: want it below the baseline %d, in the line (height %d)", u.Min.Y, u.Max.Y, baseline, l.Dims.Size.Y)
	}
}

// Wrapped text gets a line under each line, as long as that line.
func TestDecorationPerLine(t *testing.T) {
	l := laidOut(t, "a long first line wraps here then a short one", 150, gtext.Start)
	lines := decorationLines(l, properties.TextDecorationUnderline, 1)
	if len(lines) < 2 {
		t.Fatalf("%d lines for wrapped text: %v", len(lines), lines)
	}
	for i := 1; i < len(lines); i++ {
		if lines[i].Min.Y <= lines[i-1].Max.Y {
			t.Errorf("line %d at y %d isn't below line %d at %d", i, lines[i].Min.Y, i-1, lines[i-1].Max.Y)
		}
	}
	if last := lines[len(lines)-1]; last.Max.X >= lines[0].Max.X {
		t.Errorf("the short last line's underline (%v) is as long as the first's (%v)", last, lines[0])
	}
}

// Aligned to the end, the line follows the text there.
func TestDecorationAligned(t *testing.T) {
	l := laidOut(t, "Done", 400, gtext.End)
	r := decorationLines(l, properties.TextDecorationUnderline, 1)[0]
	if r.Min.X < 300 || r.Max.X > 401 {
		t.Errorf("end-aligned underline %v: want it at the right, under the text", r)
	}
}
