package base

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/state"
	"github.com/xmhrnx/giode/styles/properties"
)

func memoGtx() layout.Context {
	return layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Exact(image.Pt(200, 100)),
	}
}

// stubEl is a minimal element for cache tests: a fixed size, with an
// optional body.
type stubEl struct {
	w, h int
	fn   func(gtx layout.Context)
}

func (s stubEl) Layout(gtx layout.Context) layout.Dimensions {
	if s.fn != nil {
		s.fn(gtx)
	}
	return layout.Dimensions{Size: image.Pt(s.w, s.h)}
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
	build := countingBuild(&builds, func() Element { return stubEl{w: 10, h: 10} })

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
	build := countingBuild(&builds, func() Element { return stubEl{} })

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
		return stubEl{}
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
		return stubEl{fn: func(gtx layout.Context) {
			if animating {
				kit.MarkLive()
			}
		}}
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
	build := countingBuild(&builds, func() Element { return stubEl{} })
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
		return stubEl{fn: func(gtx layout.Context) {
			inner.Layout(gtx, func() Element {
				return stubEl{fn: func(gtx layout.Context) {
					kit.MarkLive()
				}}
			})
		}}
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
	Memo("k", func() Element { return stubEl{} }).Layout(memoGtx())
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
	build := countingBuild(&builds, func() Element { return stubEl{} })
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
