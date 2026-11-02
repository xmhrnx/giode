package virtuallist

import (
	"image"
	"testing"
	"time"

	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/elements"
)

type harness struct {
	r     input.Router
	l     *List
	scale float32
	now   time.Time
}

func (h *harness) frame() {
	var ops op.Ops
	h.now = h.now.Add(16 * time.Millisecond)
	gtx := layout.Context{
		Ops: &ops, Now: h.now, Source: h.r.Source(),
		Metric:      unit.Metric{PxPerDp: h.scale, PxPerSp: h.scale},
		Constraints: layout.Exact(image.Pt(300, int(200*h.scale))),
	}
	h.l.Layout(gtx)
	h.r.Frame(&ops)
}

func newHarness(n int, scale float32, built *[]int) *harness {
	l := New(func() int { return n }, 20, func(i int) elements.Element {
		*built = append(*built, i)
		return elements.Raw(func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: gtx.Constraints.Max}
		})
	})
	return &harness{l: l, scale: scale, now: time.Unix(0, 0)}
}

func TestOnlyVisibleRowsAreBuilt(t *testing.T) {
	var built []int
	h := newHarness(100000, 2, &built)
	h.frame()
	// 200dp viewport / 20dp rows = 10 visible, plus 3 overscan below.
	if len(built) != 13 {
		t.Errorf("built %d rows, want 13", len(built))
	}
	if f, l := h.l.VisibleRange(); f != 0 || l != 9 {
		t.Errorf("visible range = %d..%d, want 0..9", f, l)
	}
}

func TestScrollToIndexAligns(t *testing.T) {
	var built []int
	h := newHarness(1000, 1.5, &built)
	h.frame()
	cases := []struct {
		align       Align
		first, last int
	}{
		{AlignStart, 500, 509},
		{AlignEnd, 491, 500},
		{AlignCenter, 495, 505},
	}
	for _, c := range cases {
		h.l.ScrollToIndex(500, c.align)
		h.frame()
		if f, l := h.l.VisibleRange(); f != c.first || l != c.last {
			t.Errorf("align %d: visible %d..%d, want %d..%d", c.align, f, l, c.first, c.last)
		}
	}
	// Nearest doesn't move when already visible.
	h.l.ScrollToIndex(498, AlignNearest)
	h.frame()
	if f, _ := h.l.VisibleRange(); f != 495 {
		t.Errorf("nearest moved an already-visible row: first %d", f)
	}
	h.l.ScrollToIndex(5000, AlignStart)
	h.frame()
	if _, l := h.l.VisibleRange(); l != 999 {
		t.Errorf("scrolling past the end: last %d, want 999", l)
	}
}

func TestKeyboardNavigation(t *testing.T) {
	var built []int
	h := newHarness(100, 1, &built)
	var activated []int
	h.l.OnActivate(func(i int) { activated = append(activated, i) })
	h.frame()
	h.r.Source().Execute(key.FocusCmd{Tag: h.l})
	h.frame()
	if !h.l.HasFocus() || h.l.FocusedIndex() != 0 {
		t.Fatalf("after focusing: focus %v, index %d", h.l.HasFocus(), h.l.FocusedIndex())
	}
	press := func(name key.Name) {
		h.r.Queue(key.Event{Name: name, State: key.Press})
		h.frame()
		h.frame()
	}
	press(key.NameDownArrow)
	press(key.NameDownArrow)
	if h.l.FocusedIndex() != 2 {
		t.Errorf("after Down x2: %d, want 2", h.l.FocusedIndex())
	}
	press(key.NamePageDown)
	if i := h.l.FocusedIndex(); i != 11 {
		t.Errorf("after PageDown: %d, want 11", i)
	}
	if _, last := h.l.VisibleRange(); last < 11 {
		t.Errorf("focused row 11 not scrolled into view (last %d)", last)
	}
	press(key.NameEnd)
	if h.l.FocusedIndex() != 99 {
		t.Errorf("after End: %d", h.l.FocusedIndex())
	}
	press(key.NameHome)
	press(key.NameReturn)
	if h.l.FocusedIndex() != 0 || len(activated) != 1 || activated[0] != 0 {
		t.Errorf("Home + Enter: focus %d, activated %v", h.l.FocusedIndex(), activated)
	}
}

func TestOnReachEndIsEdgeTriggered(t *testing.T) {
	var built []int
	h := newHarness(50, 1, &built)
	ends := 0
	h.l.OnReachEnd(func() { ends++ })
	h.frame()
	h.l.ScrollToIndex(49, AlignEnd)
	h.frame()
	h.frame()
	if ends != 1 {
		t.Errorf("reach-end calls = %d, want 1", ends)
	}
	h.l.ScrollToIndex(0, AlignStart)
	h.frame()
	h.l.ScrollToIndex(49, AlignEnd)
	h.frame()
	if ends != 2 {
		t.Errorf("after leaving and returning: %d calls, want 2", ends)
	}
}

func TestShrinkingCountClampsFocus(t *testing.T) {
	n := 10
	var built []int
	l := New(func() int { return n }, 20, func(i int) elements.Element {
		built = append(built, i)
		return elements.Raw(func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} })
	})
	h := &harness{l: l, scale: 1, now: time.Unix(0, 0)}
	h.frame()
	l.SetFocusedIndex(9)
	n = 3
	built = nil
	h.frame()
	if l.FocusedIndex() != 2 {
		t.Errorf("focus after shrinking = %d, want 2", l.FocusedIndex())
	}
	for _, i := range built {
		if i >= 3 {
			t.Errorf("built row %d of a 3-row list", i)
		}
	}
}
