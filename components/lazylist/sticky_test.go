package lazylist

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

// headerAt lays the list out and returns where "Day n" headers show.
func headerAt(l *List) map[string][]int {
	elements.BeginFrame()
	defer elements.EndFrame()
	elements.BeginSemantics()
	var ops op.Ops
	l.Layout(layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(200, 300))})
	out := map[string][]int{}
	for _, n := range elements.EndSemantics() {
		if n.Kind == elements.SemanticText && len(n.Label) > 4 && n.Label[:4] == "Day " {
			out[n.Label] = append(out[n.Label], n.Bounds.Min.Y)
		}
	}
	return out
}

// A day header every ten 30 dp rows: the current day's stays at the
// top, far above as it is, until the next pushes it up.
func TestStickyHeaders(t *testing.T) {
	fonts.UseGofont()
	l := New(func() int { return 200 }, func(i int) any { return i }, func(i int) elements.Element {
		label := fmt.Sprintf("message %d", i)
		if i%10 == 0 {
			label = fmt.Sprintf("Day %d", i/10)
		}
		return elements.Box(styles.Styles{Height: 30, Width: -1}, elements.Text(label))
	}).StickyHeaders(func(i int) bool { return i%10 == 0 })
	headerAt(l)
	l.ScrollToKey(55, AlignStart)
	headerAt(l)
	got := headerAt(l)
	if !has(got["Day 5"], 0) {
		t.Errorf("Day 5 (row 50, five rows up) at %v, want pinned at 0", got)
	}
	// Row 59's bottom 10 dp at the top: Day 6 at 10 pushes Day 5 to -20.
	l.ScrollToKey(59, AlignStart)
	headerAt(l)
	l.ScrollBy(0, 20)
	headerAt(l)
	got = headerAt(l)
	if !has(got["Day 6"], 10) || !has(got["Day 5"], -20) || has(got["Day 5"], 0) {
		t.Errorf("pushing: %v", got)
	}
	_ = properties.Height(0)
}

func has(ys []int, y int) bool {
	for _, v := range ys {
		if v == y {
			return true
		}
	}
	return false
}
