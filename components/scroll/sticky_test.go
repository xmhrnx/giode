package scroll

import (
	"fmt"
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// sections are three 300 dp sections, each starting with a 30 dp sticky
// header.
func sections() elements.Element {
	var kids []elements.Element
	for i := range 3 {
		kids = append(kids, elements.Stack(styles.Styles{Width: -1},
			elements.Box(styles.Styles{Position: properties.PositionSticky, Height: 30, Width: -1},
				elements.Text(fmt.Sprintf("Section %d", i+1))),
			elements.Box(styles.Styles{Height: 270, Width: -1}),
		))
	}
	return elements.Stack(styles.Styles{Width: -1}, kids...)
}

// headerY lays s out at offset and returns where each header's text is.
func headerY(t *testing.T, s *Scroll, offset float32) []int {
	t.Helper()
	s.offset = offset
	elements.BeginFrame()
	defer elements.EndFrame()
	elements.BeginSemantics()
	var ops op.Ops
	s.Layout(layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(200, 200))})
	ys := make([]int, 3)
	for _, n := range elements.EndSemantics() {
		var i int
		if _, err := fmt.Sscanf(n.Label, "Section %d", &i); err == nil && n.Kind == elements.SemanticText {
			ys[i-1] = n.Bounds.Min.Y
		}
	}
	return ys
}

func TestStickyHeaders(t *testing.T) {
	fonts.UseGofont()
	s := New(sections)
	for _, c := range []struct {
		offset float32
		want   [3]int
	}{
		{0, [3]int{0, 300, 600}},     // in the flow
		{100, [3]int{0, 200, 500}},   // the first held at the top
		{285, [3]int{-15, 15, 315}},  // the second pushing it out
		{350, [3]int{-80, 0, 250}},   // the second held
		{700, [3]int{-430, -130, 0}}, // the last held to the end
	} {
		got := headerY(t, s, c.offset)
		if [3]int(got) != c.want {
			t.Errorf("offset %v: headers at %v, want %v", c.offset, got, c.want)
		}
	}
}

// Outside a Scroll a sticky box paints in the flow.
func TestStickyOutsideScroll(t *testing.T) {
	var ops op.Ops
	elements.BeginFrame()
	defer elements.EndFrame()
	dims := elements.Box(styles.Styles{Position: properties.PositionSticky, Width: 50, Height: 20}).Layout(layout.Context{Ops: &ops, Constraints: layout.Constraints{Max: image.Pt(100, 100)}})
	if dims.Size != image.Pt(50, 20) {
		t.Errorf("sticky outside a scroll: %v", dims.Size)
	}
}
