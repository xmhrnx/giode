package elements

import (
	"image"
	"testing"

	"gioui.org/io/input"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/elements/internal/base"
	"github.com/xmhrnx/giode/styles"
)

func TestRegisterPortal_DrainInvokesBuildOnce(t *testing.T) {
	SetWindowSize(image.Pt(800, 600))
	defer SetWindowSize(image.Point{})

	var ops op.Ops
	var router input.Router
	gtx := layout.Context{Ops: &ops, Source: router.Source(), Constraints: layout.Constraints{Max: image.Pt(800, 600)}}

	calls := 0
	build := func() Element {
		calls++
		return Box(styles.Styles{Width: 20, Height: 20})
	}
	RegisterPortal(gtx, image.Pt(10, 10), build)
	if calls != 1 {
		t.Fatalf("build calls after registration (scratch measure) = %d, want 1", calls)
	}

	DrainPortals(gtx)
	if calls != 2 {
		t.Fatalf("build calls after DrainPortals = %d, want 2 (one measure, one real draw)", calls)
	}

	// The queue is cleared after draining: a second drain calls
	// nothing.
	DrainPortals(gtx)
	if calls != 2 {
		t.Errorf("build calls after draining an already-empty queue = %d, want 2 (unchanged)", calls)
	}
}

func TestRegisterPortal_StacksInRegistrationOrder(t *testing.T) {
	SetWindowSize(image.Pt(800, 600))
	defer SetWindowSize(image.Point{})

	var ops op.Ops
	var router input.Router
	gtx := layout.Context{Ops: &ops, Source: router.Source(), Constraints: layout.Constraints{Max: image.Pt(800, 600)}}

	var order []int
	RegisterPortal(gtx, image.Pt(0, 0), func() Element {
		order = append(order, 1)
		return Box(styles.Styles{Width: 10, Height: 10})
	})
	RegisterPortal(gtx, image.Pt(0, 0), func() Element {
		order = append(order, 2)
		return Box(styles.Styles{Width: 10, Height: 10})
	})
	order = nil // discard the scratch-measure calls, only check the real draw order

	DrainPortals(gtx)
	if len(order) != 2 || order[0] != 1 || order[1] != 2 {
		t.Errorf("draw order = %v, want [1 2] (registration order, so later registrations paint on top)", order)
	}
}

// TestRegisterPortal_UsesWindowSizeNotAmbientConstraints guards the
// bug SetWindowSize fixes: without it, a portal triggered from deep
// inside a tightly constrained ambient box (far smaller than the real
// window - the common case for Tooltip/Dropdown, unlike Menu's own
// panel, which happens to be laid out directly off the window) would
// measure and clamp against that tiny box instead of the real window,
// positioning it more conservatively than necessary.
func TestRegisterPortal_UsesWindowSizeNotAmbientConstraints(t *testing.T) {
	SetWindowSize(image.Pt(800, 600))
	defer SetWindowSize(image.Point{})
	defer base.ClearPortals()

	var ops op.Ops
	var router input.Router
	// A caller nested inside a small, tight ambient box - far smaller
	// than the real window.
	tightGtx := layout.Context{Ops: &ops, Source: router.Source(), Constraints: layout.Constraints{Max: image.Pt(50, 50)}}

	// Anchored at x=700: comfortably inside the real 800-wide window
	// with a 60-wide panel and the default 8px margin (700+60+8=768),
	// but far outside the tight 50-wide ambient box.
	RegisterPortal(tightGtx, image.Pt(700, 10), func() Element {
		return Box(styles.Styles{Width: 60, Height: 20})
	})

	if q := base.PendingPortals(); len(q) != 1 {
		t.Fatalf("portalQueue len = %d, want 1", len(q))
	} else if pos := q[0].Pos(); pos.X < 600 {
		t.Errorf("pos.X = %d, want it left un-clamped (near 700): the real window is 800 wide, not the 50-wide ambient constraint", pos.X)
	}
}

// TestRegisterPortalAround_FlipsAboveTrigger checks a portal that
// doesn't fit below its trigger goes above the whole trigger rather
// than over it - a tooltip on a control bar at the window's bottom.
func TestRegisterPortalAround_FlipsAboveTrigger(t *testing.T) {
	SetWindowSize(image.Pt(800, 600))
	defer SetWindowSize(image.Point{})
	base.ClearPortals()
	defer base.ClearPortals()
	gtx := layout.Context{Ops: new(op.Ops), Constraints: layout.Exact(image.Pt(800, 600))}
	panel := func() Element { return Box(styles.Styles{Width: 120, Height: 30}) }

	bar := image.Rect(100, 550, 140, 590) // a button 10 px above the bottom edge
	RegisterPortalAround(gtx, bar, 6, PlaceAuto, panel)
	pos := base.PendingPortals()[0].Pos()
	if pos.Y+30 > bar.Min.Y-6 {
		t.Errorf("flipped panel at y=%d covers the trigger starting at %d", pos.Y, bar.Min.Y)
	}

	top := image.Rect(100, 20, 140, 60)
	RegisterPortalAround(gtx, top, 6, PlaceAuto, panel)
	if pos := base.PendingPortals()[1].Pos(); pos.Y != 66 || pos.X != 100 {
		t.Errorf("below: %v, want (100,66)", pos)
	}
	RegisterPortalAround(gtx, image.Rect(100, 200, 140, 240), 6, PlaceAbove, panel)
	if pos := base.PendingPortals()[2].Pos(); pos.Y != 200-6-30 {
		t.Errorf("above: %v", pos)
	}
	// Tracked anchors move the whole rectangle.
	q := base.PendingPortals()
	q[2].SetAnchor(q[2].Anchor().Add(image.Pt(0, 100)))
	if pos := base.PendingPortals()[2].Pos(); pos.Y != 300-6-30 {
		t.Errorf("moved: %v", pos)
	}
}
