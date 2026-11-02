package elements

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/state"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

func memoGtx() layout.Context {
	return layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Exact(image.Pt(200, 100)),
	}
}

// countingBuild returns a build function that counts its calls.
func countingBuild(n *int, el func() Element) func() Element {
	return func() Element {
		*n++
		return el()
	}
}

func TestMemoCache_ReusesUntilDepsChange(t *testing.T) {
	var c MemoCache
	builds := 0
	build := countingBuild(&builds, func() Element { return Box(styles.Styles{Width: 10, Height: 10}) })

	for i := 0; i < 3; i++ {
		BeginFrame()
		dims := c.Layout(memoGtx(), build, "a", 1)
		if dims.Size != image.Pt(10, 10) {
			t.Fatalf("frame %d: size = %v", i, dims.Size)
		}
	}
	if builds != 1 {
		t.Fatalf("builds after 3 identical frames = %d, want 1", builds)
	}

	BeginFrame()
	c.Layout(memoGtx(), build, "a", 2)
	if builds != 2 {
		t.Fatalf("builds after dep change = %d, want 2", builds)
	}
	if hits, misses := c.Stats(); hits != 2 || misses != 2 {
		t.Errorf("stats = %d hits, %d misses, want 2, 2", hits, misses)
	}
}

func TestMemoCache_RebuildsOnConstraintChange(t *testing.T) {
	var c MemoCache
	builds := 0
	build := countingBuild(&builds, func() Element { return Box(styles.Styles{}) })

	BeginFrame()
	c.Layout(memoGtx(), build)
	BeginFrame()
	gtx := memoGtx()
	gtx.Constraints = layout.Exact(image.Pt(300, 100))
	c.Layout(gtx, build)
	if builds != 2 {
		t.Fatalf("builds = %d, want 2", builds)
	}
}

func TestMemoCache_TracksStateReads(t *testing.T) {
	var c MemoCache
	s := state.New("hello")
	builds := 0
	build := countingBuild(&builds, func() Element {
		_ = s.Get()
		return Box(styles.Styles{})
	})

	BeginFrame()
	c.Layout(memoGtx(), build)
	BeginFrame()
	c.Layout(memoGtx(), build)
	if builds != 1 {
		t.Fatalf("builds before state change = %d, want 1", builds)
	}
	s.Set("world")
	BeginFrame()
	c.Layout(memoGtx(), build)
	if builds != 2 {
		t.Fatalf("builds after state change = %d, want 2", builds)
	}
}

func TestMemoCache_LiveWidgetForcesRebuild(t *testing.T) {
	var c MemoCache
	builds := 0
	animating := true
	build := countingBuild(&builds, func() Element {
		return Raw(func(gtx layout.Context) layout.Dimensions {
			if animating {
				kit.MarkLive()
			}
			return layout.Dimensions{}
		})
	})
	for i := 0; i < 3; i++ {
		BeginFrame()
		c.Layout(memoGtx(), build)
	}
	if builds != 3 {
		t.Fatalf("builds while live = %d, want 3", builds)
	}
	animating = false
	BeginFrame()
	c.Layout(memoGtx(), build) // last live frame settles
	BeginFrame()
	c.Layout(memoGtx(), build)
	if builds != 4 {
		t.Fatalf("builds after settling = %d, want 4", builds)
	}
}

func TestMemoCache_ThemeChangeRebuilds(t *testing.T) {
	t.Cleanup(func() { properties.SetMode(properties.ModeLight) })
	var c MemoCache
	builds := 0
	build := countingBuild(&builds, func() Element { return Box(styles.Styles{Background: "theme.surface"}) })
	BeginFrame()
	c.Layout(memoGtx(), build)
	properties.SetMode(properties.ModeDark)
	BeginFrame()
	c.Layout(memoGtx(), build)
	if builds != 2 {
		t.Fatalf("builds after theme change = %d, want 2", builds)
	}
}

func TestMemoCache_NestedLiveMarksParent(t *testing.T) {
	var outer, inner MemoCache
	outerBuilds := 0
	build := countingBuild(&outerBuilds, func() Element {
		return Raw(func(gtx layout.Context) layout.Dimensions {
			return inner.Layout(gtx, func() Element {
				return Raw(func(gtx layout.Context) layout.Dimensions {
					kit.MarkLive()
					return layout.Dimensions{}
				})
			})
		})
	})
	BeginFrame()
	outer.Layout(memoGtx(), build)
	BeginFrame()
	outer.Layout(memoGtx(), build)
	if outerBuilds != 2 {
		t.Fatalf("outer builds = %d, want 2 (live child must keep parent live)", outerBuilds)
	}
}

func TestMemo_KeyedEviction(t *testing.T) {
	BeginFrame()
	Memo("k", func() Element { return Box(styles.Styles{}) }).Layout(memoGtx())
	EndFrame()
	if _, ok := memos["k"]; !ok {
		t.Fatal("memo evicted in the frame it was used")
	}
	BeginFrame()
	EndFrame()
	if _, ok := memos["k"]; ok {
		t.Fatal("unused memo not evicted")
	}
}

func TestMemoCache_DoubleUseFallsBack(t *testing.T) {
	var c MemoCache
	builds := 0
	build := countingBuild(&builds, func() Element { return Box(styles.Styles{}) })
	BeginFrame()
	c.Layout(memoGtx(), build)
	c.Layout(memoGtx(), build)
	if builds != 2 {
		t.Fatalf("builds = %d, want 2 (second use in a frame is uncached)", builds)
	}
}

func TestDepsEqual_Uncomparable(t *testing.T) {
	if !depsEqual([]any{[]int{1, 2}}, []any{[]int{1, 2}}) {
		t.Error("equal slices reported different")
	}
	if depsEqual([]any{[]int{1, 2}}, []any{[]int{1, 3}}) {
		t.Error("different slices reported equal")
	}
	if depsEqual([]any{1}, []any{int64(1)}) {
		t.Error("different types reported equal")
	}
}

func TestSemantics_FlexChildBounds(t *testing.T) {
	BeginSemantics()
	gtx := layout.Context{Ops: new(op.Ops), Constraints: layout.Constraints{Max: image.Pt(400, 400)}}
	sized := func(w, h int, label string) Element {
		return Raw(func(gtx layout.Context) layout.Dimensions {
			RecordSemantic(gtx, SemanticNode{Kind: SemanticCustom, Label: label}, image.Pt(w, h))
			return layout.Dimensions{Size: image.Pt(w, h)}
		})
	}
	Row(styles.Styles{Gap: 10, Padding: styles.Styles{}.Padding},
		sized(50, 20, "a"),
		Stack(styles.Styles{Gap: 5}, sized(30, 10, "b"), sized(30, 10, "c")),
	).Layout(gtx)
	nodes := EndSemantics()
	want := map[string]image.Point{"a": {0, 0}, "b": {60, 0}, "c": {60, 15}}
	for _, n := range nodes {
		if got := n.Bounds.Min; got != want[n.Label] {
			t.Errorf("%s at %v, want %v", n.Label, got, want[n.Label])
		}
	}
	if len(nodes) != 3 {
		t.Fatalf("nodes = %d, want 3", len(nodes))
	}
}

func TestRow_AlignCenterDoesNotStretchChildren(t *testing.T) {
	BeginSemantics()
	gtx := layout.Context{Ops: new(op.Ops), Constraints: layout.Constraints{Max: image.Pt(400, 400)}}
	child := Raw(func(gtx layout.Context) layout.Dimensions {
		size := image.Pt(20, max(10, gtx.Constraints.Min.Y))
		RecordSemantic(gtx, SemanticNode{Kind: SemanticCustom, Label: "c"}, size)
		return layout.Dimensions{Size: size}
	})
	for _, wrap := range []properties.Wrap{properties.WrapNoWrap, properties.WrapWrap} {
		BeginSemantics()
		dims := Row(styles.Styles{Height: 40, Align: properties.AlignCenter, Wrap: wrap}, child).Layout(gtx)
		n := EndSemantics()[0]
		if dims.Size.Y != 40 {
			t.Errorf("wrap %q: row height = %d, want 40", wrap, dims.Size.Y)
		}
		if n.Bounds.Dy() != 10 || n.Bounds.Min.Y != 15 {
			t.Errorf("wrap %q: child at %v, want 10 tall centered at y=15", wrap, n.Bounds)
		}
	}
	EndSemantics()
}

func TestRow_FlexGrowBoxFills(t *testing.T) {
	BeginSemantics()
	gtx := layout.Context{Ops: new(op.Ops), Constraints: layout.Constraints{Max: image.Pt(400, 100)}}
	fixed := func(w int, label string) Element {
		return Raw(func(gtx layout.Context) layout.Dimensions {
			RecordSemantic(gtx, SemanticNode{Kind: SemanticCustom, Label: label}, image.Pt(w, 10))
			return layout.Dimensions{Size: image.Pt(w, 10)}
		})
	}
	Row(styles.Styles{Width: -1, Wrap: properties.WrapNoWrap},
		Box(styles.Styles{FlexGrow: 1}, fixed(50, "grow")),
		fixed(60, "end"),
	).Layout(gtx)
	for _, n := range EndSemantics() {
		if n.Label == "end" && n.Bounds.Min.X != 340 {
			t.Errorf("item after a FlexGrow box at x=%d, want 340 (pushed to the end)", n.Bounds.Min.X)
		}
	}
}
