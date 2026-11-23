package textrender

import (
	"image"
	"strings"
	"testing"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/fonts"
)

func TestLayoutRich_MaxLines(t *testing.T) {
	fonts.UseGofont()
	sh := fonts.Shaper()
	spans := []Span{
		{Size: 15, Content: "Run "},
		{Size: 15, Font: font.Font{Weight: font.Bold}, Content: "everything"},
		{Size: 15, Content: " and read the " + strings.Repeat("documentation carefully ", 8)},
	}
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: layout.Constraints{Max: image.Pt(200, 1000)}, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
	all := layoutRich(sh, gtx, RichParams{}, spans)
	two := layoutRich(sh, gtx, RichParams{MaxLines: 2, Ellipsis: true}, spans)
	tops := map[int]bool{}
	for _, s := range two.Segments {
		tops[s.Offset.Y] = true
		if s.Offset.X+s.Size.X > 200 {
			t.Errorf("segment reaches %d, past the width", s.Offset.X+s.Size.X)
		}
	}
	if len(tops) != 2 {
		t.Errorf("%d lines, want 2", len(tops))
	}
	if two.Dims.Size.Y >= all.Dims.Size.Y {
		t.Errorf("height %d, want less than the full %d", two.Dims.Size.Y, all.Dims.Size.Y)
	}
	// The last segment is the ellipsis.
	last := two.Segments[len(two.Segments)-1]
	if last.RuneStart != last.RuneEnd || len(last.Glyphs) == 0 {
		t.Errorf("last segment %+v isn't the ellipsis", last)
	}
	// Text that fits gets no ellipsis.
	short := layoutRich(sh, gtx, RichParams{MaxLines: 2}, spans[:2])
	if l := short.Segments[len(short.Segments)-1]; l.RuneStart == l.RuneEnd {
		t.Error("an ellipsis on text that fits")
	}
}
