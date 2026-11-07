package menu

import (
	"image"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

type ctxHarness struct {
	r    input.Router
	now  time.Time
	view func() elements.Element
}

func (h *ctxHarness) frame() {
	elements.BeginFrame()
	defer elements.EndFrame()
	h.now = h.now.Add(16 * time.Millisecond)
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Now: h.now, Source: h.r.Source(), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(600, 400))}
	elements.SetWindowSize(gtx.Constraints.Max)
	h.view().Layout(gtx)
	elements.DrainContextMenus(gtx)
	elements.DrainPortals(gtx)
	h.r.Frame(&ops)
}

func (h *ctxHarness) rightClick(x, y float32) {
	p := f32.Pt(x, y)
	h.r.Queue(
		pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: p},
		pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonSecondary, Position: p},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: p},
	)
	h.frame()
	h.frame()
}

// A context menu opens at the pointer, on a small element deep in the
// layout; with elements inside each other, the innermost with a menu
// opens its own.
func TestContextOpensAtPointer(t *testing.T) {
	defer elements.SetWindowSize(image.Point{})
	outerMenu := New(styles.Styles{}).Item("Outer", func() {})
	innerMenu := New(styles.Styles{}).Item("Inner", func() {})
	defer outerMenu.Close()
	defer innerMenu.Close()
	var outer, inner *elements.BoxElement
	h := &ctxHarness{now: time.Unix(0, 0), view: func() elements.Element {
		// Built every frame, as views are.
		inner = elements.Box(styles.Styles{Width: 40, Height: 20})
		inner.Context(innerMenu)
		outer = elements.Box(styles.Styles{Width: 200, Height: 100, Padding: properties.UniformInset(30)}, inner)
		outer.Context(outerMenu)
		return elements.Stack(styles.Styles{Padding: properties.UniformInset(50), Gap: 10},
			elements.Box(styles.Styles{Width: 10, Height: 70}),
			elements.Row(styles.Styles{Gap: 100}, elements.Box(styles.Styles{Width: 60, Height: 10}), outer),
		)
	}}
	h.frame()
	h.frame()
	// The inner box: the row is at (50, 130); outer at x 50+60+100 = 210,
	// inner at (240, 160) to (280, 180).
	h.rightClick(255, 170)
	if !innerMenu.Opened() || outerMenu.Opened() {
		t.Fatalf("right-click on the inner box: inner open %v, outer open %v", innerMenu.Opened(), outerMenu.Opened())
	}
	if innerMenu.pos != f32.Pt(255, 170) {
		t.Errorf("the inner menu opened at %v, want the pointer (255, 170)", innerMenu.pos)
	}
	innerMenu.Close()
	h.frame()
	// The outer box only.
	h.rightClick(390, 215)
	if !outerMenu.Opened() || innerMenu.Opened() {
		t.Fatalf("right-click on the outer box: outer open %v, inner open %v", outerMenu.Opened(), innerMenu.Opened())
	}
	if outerMenu.pos != f32.Pt(390, 215) {
		t.Errorf("the outer menu opened at %v, want the pointer (390, 215)", outerMenu.pos)
	}
	outerMenu.Close()
	h.frame()
	// A left click opens nothing.
	p := f32.Pt(255, 170)
	h.r.Queue(
		pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: p},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: p},
	)
	h.frame()
	if innerMenu.Opened() || outerMenu.Opened() {
		t.Error("a left click opened a context menu")
	}
}

// A finger held still on the element opens its menu where it rests; the
// menu draws through the frame, its items in the semantic tree.
func TestContextLongPress(t *testing.T) {
	defer elements.SetWindowSize(image.Point{})
	m := New(styles.Styles{}).Item("Rename", func() {})
	defer m.Close()
	btn := elements.Box(styles.Styles{Width: 120, Height: 40})
	btn.Context(m)
	h := &ctxHarness{now: time.Unix(0, 0), view: func() elements.Element {
		return elements.Stack(styles.Styles{Padding: properties.UniformInset(100)}, btn)
	}}
	h.frame()
	h.frame()
	p := f32.Pt(130, 115)
	h.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Touch, Position: p, PointerID: 1})
	h.frame()
	if m.Opened() {
		t.Fatal("the menu opened before the long-press")
	}
	for i := 0; i < 40; i++ {
		h.frame()
	}
	if !m.Opened() || m.pos != p {
		t.Fatalf("after holding: open %v at %v, want open at %v", m.Opened(), m.pos, p)
	}
	elements.BeginSemantics()
	h.frame()
	found := false
	for _, n := range elements.EndSemantics() {
		if n.Label == "Rename" {
			found = true
		}
	}
	if !found {
		t.Error("the open menu isn't drawn")
	}
}

// An element replayed from a cache - a Memo here - still opens its menu
// at the pointer.
func TestContextInCache(t *testing.T) {
	defer elements.SetWindowSize(image.Point{})
	m := New(styles.Styles{}).Item("Delete", func() {})
	defer m.Close()
	row := elements.Box(styles.Styles{Width: 300, Height: 30})
	row.Context(m)
	h := &ctxHarness{now: time.Unix(0, 0), view: func() elements.Element {
		return elements.Stack(styles.Styles{Padding: properties.UniformInset(40)},
			elements.Memo("rows", func() elements.Element { return row }))
	}}
	for i := 0; i < 4; i++ {
		h.frame()
	}
	h.rightClick(100, 50)
	if !m.Opened() || m.pos != f32.Pt(100, 50) {
		t.Errorf("in a cached subtree: open %v at %v, want open at (100, 50)", m.Opened(), m.pos)
	}
}
