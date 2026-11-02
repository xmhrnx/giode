package toaster

import (
	"image"
	"sync"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/styles/properties"
)

func init() {
	fonts.UseGofont()
}

type harness struct {
	r   input.Router
	t   *Toaster
	now time.Time
}

// frame lays the toaster out the way the app does, portals included,
// and returns the semantic nodes.
func (h *harness) frame(advance time.Duration) []elements.SemanticNode {
	h.now = h.now.Add(advance)
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Now: h.now, Constraints: layout.Exact(image.Pt(800, 600)), Source: h.r.Source()}
	elements.BeginFrame()
	elements.BeginSemantics()
	elements.SetWindowSize(gtx.Constraints.Max)
	h.t.Layout(gtx)
	elements.DrainPortals(gtx)
	nodes := elements.EndSemantics()
	elements.EndFrame()
	h.r.Frame(&ops)
	return nodes
}

func labels(nodes []elements.SemanticNode, kind elements.SemanticKind) []string {
	var out []string
	for _, n := range nodes {
		if n.Kind == kind {
			out = append(out, n.Label)
		}
	}
	return out
}

func TestShowExpireAndMax(t *testing.T) {
	h := &harness{t: New().Duration(time.Second).Max(2), now: time.Unix(0, 0)}
	h.t.Show(Info, "one")
	h.t.Show(Success, "two")
	h.t.Show(Error, "three")
	h.frame(0)
	nodes := h.frame(300 * time.Millisecond)
	if got := labels(nodes, elements.SemanticText); len(got) != 2 || got[0] != "one" || got[1] != "two" {
		t.Fatalf("visible toasts = %v, want the first two (Max 2)", got)
	}
	// After the first two expire and animate out, the third shows.
	h.frame(time.Second)
	h.frame(300 * time.Millisecond)
	h.frame(300 * time.Millisecond)
	nodes = h.frame(100 * time.Millisecond)
	if got := labels(nodes, elements.SemanticText); len(got) != 1 || got[0] != "three" {
		t.Fatalf("after expiry = %v, want [three]", got)
	}
	h.frame(2 * time.Second)
	h.frame(400 * time.Millisecond)
	h.frame(400 * time.Millisecond)
	if h.t.Count() != 0 {
		t.Errorf("toasts left after everything expired: %d", h.t.Count())
	}
}

func TestHoverPausesExpiry(t *testing.T) {
	h := &harness{t: New().Duration(time.Second), now: time.Unix(0, 0)}
	h.t.Show(Info, "hover me")
	h.frame(0)
	nodes := h.frame(300 * time.Millisecond)
	var bounds image.Rectangle
	for _, n := range nodes {
		if n.Kind == elements.SemanticText {
			bounds = n.Bounds
		}
	}
	if bounds.Empty() {
		t.Fatal("toast text not found")
	}
	if bounds.Min.X < 800-340-16 || bounds.Max.Y < 560 {
		t.Errorf("bottom-right toast at %v", bounds)
	}
	c := bounds.Min.Add(image.Pt(2, 2))
	h.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: f32.Pt(float32(c.X), float32(c.Y))})
	h.frame(10 * time.Millisecond)
	h.frame(5 * time.Second)
	if h.t.Count() != 1 {
		t.Fatal("hovered toast expired")
	}
	h.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: f32.Pt(10, 10)})
	h.frame(10 * time.Millisecond)
	h.frame(2 * time.Second)
	h.frame(400 * time.Millisecond)
	h.frame(400 * time.Millisecond)
	if h.t.Count() != 0 {
		t.Errorf("toast still up %d after unhovering past its deadline", h.t.Count())
	}
}

func TestActionAndDismiss(t *testing.T) {
	h := &harness{t: New(), now: time.Unix(0, 0)}
	ran := 0
	h.t.ShowAction(Warning, "Deleted", "Undo", func() { ran++ })
	id := h.t.Show(Info, "other")
	h.frame(0)
	nodes := h.frame(300 * time.Millisecond)
	for _, n := range nodes {
		if n.Label == "Undo" && n.Kind == elements.SemanticButton {
			n.Target.(interface{ Click() }).Click()
		}
	}
	h.t.Dismiss(id)
	h.frame(10 * time.Millisecond)
	h.frame(400 * time.Millisecond)
	h.frame(400 * time.Millisecond)
	if ran != 1 || h.t.Count() != 0 {
		t.Errorf("action ran %d times, %d toasts left", ran, h.t.Count())
	}
}

func TestShowFromGoroutines(t *testing.T) {
	h := &harness{t: New().Max(100), now: time.Unix(0, 0)}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			h.t.Show(Info, "x")
		}()
	}
	wg.Wait()
	h.frame(0)
	if h.t.Count() != 20 {
		t.Errorf("toasts = %d, want 20", h.t.Count())
	}
}

// TestInsetKeepsToastsClearOfBars checks Inset moves toasts away from
// the edges: clear of a 60 dp control bar at the bottom.
func TestInsetKeepsToastsClearOfBars(t *testing.T) {
	bottom := func(tst *Toaster) int {
		h := &harness{t: tst, now: time.Unix(0, 0)}
		h.t.Show(Info, "saved")
		h.frame(0)
		nodes := h.frame(time.Second)
		for _, n := range nodes {
			if n.Kind == elements.SemanticText && n.Label == "saved" {
				return n.Bounds.Max.Y
			}
		}
		t.Fatal("toast not shown")
		return 0
	}
	plain := bottom(New())
	inset := bottom(New().Inset(properties.Inset{Bottom: 60}))
	if plain-inset != 60 {
		t.Errorf("toast bottom %d with Inset, %d without: want 60 px higher", inset, plain)
	}
}

func TestPlacementFollowsDirection(t *testing.T) {
	for _, c := range []struct {
		p        Placement
		rtl      bool
		physical Placement
	}{
		{BottomEnd, false, BottomRight}, {BottomEnd, true, BottomLeft},
		{TopStart, false, TopLeft}, {TopStart, true, TopRight},
		{BottomRight, true, BottomRight}, {TopCenter, true, TopCenter},
	} {
		if got := c.p.physical(c.rtl); got != c.physical {
			t.Errorf("%d rtl=%v: %d, want %d", c.p, c.rtl, got, c.physical)
		}
	}
	if New().placement != BottomEnd {
		t.Error("the default placement isn't BottomEnd")
	}
}
