package elements

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/elements/internal/base"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// portalProbe registers a portal anchored at its own position, as a
// dropdown does, and tracks that position.
type portalProbe struct {
	w, h int
	pos  image.Point
}

func (p *portalProbe) Layout(gtx layout.Context) layout.Dimensions {
	TrackPosition(&p.pos)
	RegisterPortal(gtx, AbsolutePosition(), func() Element { return Box(styles.Styles{Width: 10, Height: 10}) })
	return layout.Dimensions{Size: image.Pt(p.w, p.h)}
}

func layoutFrame(el Element, max image.Point) {
	SetWindowSize(image.Pt(1000, 1000))
	BeginFrame()
	var ops op.Ops
	el.Layout(layout.Context{Ops: &ops, Constraints: layout.Constraints{Max: max}})
}

func takePortals() {
	base.ClearPortals()
}

func TestPlacement_PortalsFollowFlexChildren(t *testing.T) {
	defer SetWindowSize(image.Point{})
	a, b, c := &portalProbe{w: 100, h: 20}, &portalProbe{w: 50, h: 40}, &portalProbe{w: 30, h: 20}
	row := Row(styles.Styles{Gap: 10, Align: properties.AlignCenter, Padding: properties.UniformInset(5)}, a, b, c)
	layoutFrame(row, image.Pt(800, 600))
	q := base.PendingPortals()
	takePortals()
	if len(q) != 3 {
		t.Fatalf("%d portals, want 3", len(q))
	}
	// Padding 5; children at x 5, 115, 175; centered in a 40-tall line.
	want := []image.Point{{5, 15}, {115, 5}, {175, 15}}
	for i, w := range want {
		if q[i].Anchor() != w {
			t.Errorf("portal %d anchored at %v, want %v", i, q[i].Anchor(), w)
		}
	}
	if b.pos != want[1] || c.pos != want[2] {
		t.Errorf("tracked positions %v %v, want %v %v", b.pos, c.pos, want[1], want[2])
	}
}

func TestPlacement_NestedColumnInRow(t *testing.T) {
	defer SetWindowSize(image.Point{})
	p := &portalProbe{w: 40, h: 30}
	col := Stack(styles.Styles{}, &portalProbe{w: 40, h: 25}, p)
	row := Row(styles.Styles{}, &portalProbe{w: 70, h: 10}, col)
	layoutFrame(row, image.Pt(800, 600))
	takePortals()
	if want := image.Pt(70, 25); p.pos != want {
		t.Errorf("nested position %v, want %v", p.pos, want)
	}
}

func TestPlacement_WrappedRow(t *testing.T) {
	defer SetWindowSize(image.Point{})
	a, b := &portalProbe{w: 60, h: 20}, &portalProbe{w: 60, h: 20}
	row := Row(styles.Styles{Width: 100, Wrap: properties.WrapWrap, Gap: 4}, a, b)
	layoutFrame(row, image.Pt(800, 600))
	takePortals()
	if want := image.Pt(0, 24); b.pos != want {
		t.Errorf("wrapped item at %v, want %v", b.pos, want)
	}
}

func TestPlacement_MemoReplayRestoresProbes(t *testing.T) {
	defer SetWindowSize(image.Point{})
	p := &portalProbe{w: 40, h: 30}
	inner := Raw(func(gtx layout.Context) layout.Dimensions {
		TrackPosition(&p.pos)
		return layout.Dimensions{Size: image.Pt(40, 30)}
	})
	cache := NewMemoCache()
	cached := Raw(func(gtx layout.Context) layout.Dimensions {
		return cache.Layout(gtx, func() Element { return inner })
	})
	row := Row(styles.Styles{}, &portalProbe{w: 70, h: 10}, cached)
	for i := 0; i < 2; i++ {
		p.pos = image.Point{}
		layoutFrame(row, image.Pt(800, 600))
		takePortals()
		if want := image.Pt(70, 0); p.pos != want {
			t.Errorf("frame %d: position %v, want %v", i, p.pos, want)
		}
	}
	if hits, _ := cache.Stats(); hits != 1 {
		t.Errorf("cache hits = %d, want 1 (the second frame replays)", hits)
	}
}
