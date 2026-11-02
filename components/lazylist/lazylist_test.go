package lazylist

import (
	"fmt"
	"image"
	"sort"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
)

// model is a list of rows with keys and heights.
type model struct {
	keys    []int
	heights map[int]int
	builds  map[int]int
	version map[int]int
}

func newModel(n int) *model {
	m := &model{heights: map[int]int{}, builds: map[int]int{}, version: map[int]int{}}
	for i := 0; i < n; i++ {
		m.keys = append(m.keys, i)
		m.heights[i] = 20 + (i%5)*10
	}
	return m
}

func (m *model) list() *List {
	return New(func() int { return len(m.keys) }, func(i int) any { return m.keys[i] }, func(i int) elements.Element {
		k := m.keys[i]
		m.builds[k]++
		return elements.Box(styles.Styles{Width: -1, Height: styles.Styles{}.Height + 0, MinHeight: 0}, elements.Box(styles.Styles{Height: 0}))
	})
}

type harness struct {
	r   input.Router
	l   *List
	m   *model
	ops op.Ops
}

func newHarness(m *model, anchor Anchor, versioned bool) *harness {
	h := &harness{m: m}
	h.l = New(func() int { return len(m.keys) }, func(i int) any { return m.keys[i] }, func(i int) elements.Element {
		k := m.keys[i]
		m.builds[k]++
		return elements.Raw(func(gtx layout.Context) layout.Dimensions {
			elements.RecordSemantic(gtx, elements.SemanticNode{Kind: elements.SemanticText, Label: fmt.Sprint(k)}, image.Pt(gtx.Constraints.Max.X, m.heights[k]))
			return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, m.heights[k])}
		})
	}).Anchor(anchor)
	if versioned {
		h.l.Version(func(i int) any { return m.version[m.keys[i]]*1000 + m.heights[m.keys[i]] })
	}
	return h
}

func (h *harness) frame() []elements.SemanticNode {
	elements.BeginFrame()
	elements.BeginSemantics()
	h.ops.Reset()
	gtx := layout.Context{Ops: &h.ops, Source: h.r.Source(), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(300, 400))}
	h.l.Layout(gtx)
	nodes := elements.EndSemantics()
	h.r.Frame(&h.ops)
	elements.EndFrame()
	return nodes
}

// visible returns the keys of rows in the viewport and the top of the
// first.
func visible(nodes []elements.SemanticNode) (keys []string, firstTop int) {
	firstTop = -1
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].Bounds.Min.Y < nodes[j].Bounds.Min.Y })
	for _, n := range nodes {
		if n.Kind != elements.SemanticText || n.Bounds.Max.Y <= 0 || n.Bounds.Min.Y >= 400 {
			continue
		}
		if len(keys) == 0 {
			firstTop = n.Bounds.Min.Y
		}
		keys = append(keys, n.Label)
	}
	return keys, firstTop
}

func TestLazyList_BuildsOnlyVisibleRows(t *testing.T) {
	m := newModel(10000)
	h := newHarness(m, AnchorTop, false)
	nodes := h.frame()
	keys, top := visible(nodes)
	if len(keys) == 0 || keys[0] != "0" || top != 0 {
		t.Fatalf("first frame shows %v from %d", keys, top)
	}
	total := 0
	for _, n := range m.builds {
		total += n
	}
	// 400px viewport + 300px overscan of 20-60px rows.
	if total > 40 {
		t.Errorf("built %d rows for one frame of a 10000 row list", total)
	}
}

func TestLazyList_ScrollsToTheRightRow(t *testing.T) {
	m := newModel(1000)
	h := newHarness(m, AnchorTop, false)
	h.frame()
	h.l.ScrollBy(0, 1000)
	keys, top := visible(h.frame())
	// Rows cycle 20,30,40,50,60 = 200px per 5 rows: 1000px is row 25.
	if keys[0] != "25" || top != 0 {
		t.Errorf("after scrolling 1000px: first row %s at %d, want 25 at 0", keys[0], top)
	}
	h.l.ScrollBy(0, 15)
	keys, top = visible(h.frame())
	if keys[0] != "25" || top != -15 {
		t.Errorf("after 15px more: first row %s at %d, want 25 at -15", keys[0], top)
	}
	// A jump far down doesn't lay out the rows in between.
	clear(m.builds)
	h.l.ScrollBy(0, 100000)
	keys, _ = visible(h.frame())
	// 100000px is past the ~40000px of content: the list stops at the end.
	if last := keys[len(keys)-1]; last != "999" {
		t.Errorf("after 100000px more: last row %s, want 999", last)
	}
	if len(m.builds) > 40 {
		t.Errorf("a long jump built %d rows", len(m.builds))
	}
}

func TestLazyList_KeepsPlaceWhenRowsAboveResize(t *testing.T) {
	m := newModel(1000)
	h := newHarness(m, AnchorTop, true)
	h.frame()
	h.l.ScrollBy(0, 2005)
	before, top := visible(h.frame())
	for k := 0; k < 40; k++ {
		m.heights[k] = 100
	}
	after, top2 := visible(h.frame())
	if before[0] != after[0] || top != top2 {
		t.Errorf("rows above resized: first row %s at %d, was %s at %d", after[0], top2, before[0], top)
	}
}

func TestLazyList_KeepsPlaceWhenRowsArePrepended(t *testing.T) {
	m := newModel(200)
	h := newHarness(m, AnchorTop, false)
	h.frame()
	h.l.ScrollBy(0, 500)
	before, top := visible(h.frame())
	older := []int{}
	for k := -50; k < 0; k++ {
		older = append(older, k)
		m.heights[k] = 33
	}
	m.keys = append(older, m.keys...)
	after, top2 := visible(h.frame())
	if before[0] != after[0] || top != top2 {
		t.Errorf("rows prepended: first row %s at %d, was %s at %d", after[0], top2, before[0], top)
	}
}

func TestLazyList_BottomAnchor(t *testing.T) {
	m := newModel(100)
	h := newHarness(m, AnchorBottom, false)
	nodes := h.frame()
	keys, _ := visible(nodes)
	if keys[len(keys)-1] != "99" || !h.l.Following() {
		t.Fatalf("starts at %v, following %v", keys, h.l.Following())
	}
	bottomOf := func(nodes []elements.SemanticNode, key string) int {
		for _, n := range nodes {
			if n.Label == key {
				return n.Bounds.Max.Y
			}
		}
		return -1
	}
	if b := bottomOf(nodes, "99"); b != 400 {
		t.Errorf("last row ends at %d, want 400", b)
	}
	// A new message and a growing one keep the end in view.
	m.keys = append(m.keys, 100)
	m.heights[100] = 30
	nodes = h.frame()
	if b := bottomOf(nodes, "100"); b != 400 {
		t.Errorf("appended row ends at %d, want 400", b)
	}
	m.heights[100] = 90
	if b := bottomOf(h.frame(), "100"); b != 400 {
		t.Errorf("grown last row ends at %d, want 400", b)
	}
	// Scrolled up: appending doesn't move the view.
	h.l.ScrollBy(0, -300)
	before, top := visible(h.frame())
	if h.l.Following() {
		t.Fatal("still following after scrolling up")
	}
	m.keys = append(m.keys, 101)
	m.heights[101] = 40
	after, top2 := visible(h.frame())
	if before[0] != after[0] || top != top2 {
		t.Errorf("append while scrolled up moved the view: %s at %d, was %s at %d", after[0], top2, before[0], top)
	}
	// Back at the end, it follows again.
	h.l.ScrollBy(0, 10000)
	h.frame()
	if !h.l.Following() {
		t.Error("not following after scrolling back to the end")
	}
}

func TestLazyList_VersionedRowsReplay(t *testing.T) {
	m := newModel(100)
	h := newHarness(m, AnchorTop, true)
	h.frame()
	h.frame()
	built := m.builds[0]
	for i := 0; i < 5; i++ {
		h.frame()
	}
	if m.builds[0] != built {
		t.Errorf("unchanged row rebuilt %d times", m.builds[0]-built)
	}
	m.version[0]++
	h.frame()
	if m.builds[0] != built+1 {
		t.Errorf("row with a new version built %d times, want once", m.builds[0]-built)
	}
}

func TestLazyList_ScrollToKey(t *testing.T) {
	m := newModel(1000)
	h := newHarness(m, AnchorTop, false)
	h.frame()
	h.l.ScrollToKey(500, AlignStart)
	keys, top := visible(h.frame())
	if keys[0] != "500" || top != 0 {
		t.Errorf("ScrollToKey: first row %s at %d", keys[0], top)
	}
}

// touch queues a touch event at y and lays out a frame.
func (h *harness) touch(kind pointer.Kind, y float32) []elements.SemanticNode {
	h.r.Queue(pointer.Event{Kind: kind, Source: pointer.Touch, Position: f32.Pt(150, y), PointerID: 1})
	return h.frame()
}

func TestLazyList_FingerScrollsUpWhileLastRowGrows(t *testing.T) {
	m := newModel(100)
	h := newHarness(m, AnchorBottom, false)
	h.frame()
	h.frame()
	grow := func() { m.heights[99] += 30 }

	// The finger goes down and first moves a little up - towards the
	// end - then down, scrolling up; the reply keeps streaming.
	grow()
	h.touch(pointer.Press, 200)
	grow()
	h.touch(pointer.Move, 190) // past the slop: the list grabs
	grow()
	h.touch(pointer.Move, 185)
	if h.l.Following() {
		t.Fatal("still following once a finger moves the rows")
	}
	grow()
	h.touch(pointer.Move, 260)
	before, top := visible(h.frame())
	// Resting, then moving a little, while the last row grows.
	for i := 0; i < 5; i++ {
		grow()
		keys, top2 := visible(h.frame())
		if keys[0] != before[0] || top2 != top {
			t.Fatalf("frame %d: view moved to %s at %d, was %s at %d", i, keys[0], top2, before[0], top)
		}
	}
	grow()
	keys, top2 := visible(h.touch(pointer.Move, 280))
	if keys[0] != before[0] || top2 != top+20 {
		t.Errorf("finger moved 20px: %s at %d, want %s at %d", keys[0], top2, before[0], top+20)
	}
	grow()
	h.touch(pointer.Release, 280)
	h.frame()
	if h.l.Following() {
		t.Error("following after letting go above the end")
	}

	// Dragged back to the end and let go there: following again.
	h.touch(pointer.Press, 100)
	h.touch(pointer.Move, 90)
	h.touch(pointer.Move, -5000)
	h.touch(pointer.Release, -5000)
	for i := 0; i < 3; i++ {
		grow()
		h.frame()
	}
	if !h.l.Following() {
		t.Error("not following after letting go at the end")
	}
}
