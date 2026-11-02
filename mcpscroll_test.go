package giode

import (
	"image"
	"testing"
	"time"

	"github.com/xmhrnx/giode/elements"
)

func TestRevealStep(t *testing.T) {
	win := image.Rect(0, 0, 400, 800)
	outer := elements.SemanticNode{ID: "scroll-1", Kind: elements.SemanticScroll, Bounds: image.Rect(0, 100, 400, 700)}
	inner := elements.SemanticNode{ID: "scroll-2", Kind: elements.SemanticScroll, Bounds: image.Rect(0, 900, 400, 1100), Scroll: "scroll-1"}
	below := elements.SemanticNode{ID: "b", Bounds: image.Rect(0, 750, 100, 790), Scroll: "scroll-1"}
	nested := elements.SemanticNode{ID: "n", Bounds: image.Rect(0, 950, 100, 980), Scroll: "scroll-2"}
	deep := elements.SemanticNode{ID: "d", Bounds: image.Rect(0, 1300, 100, 1340), Scroll: "scroll-2"}
	shown := elements.SemanticNode{ID: "s", Bounds: image.Rect(0, 200, 100, 240), Scroll: "scroll-1"}
	nodes := []elements.SemanticNode{below, nested, deep, shown, inner, outer}

	if _, _, ok := revealStep(nodes, shown, win); ok {
		t.Error("a node inside the viewport needs no scrolling")
	}
	if c, d, ok := revealStep(nodes, below, win); !ok || c.ID != "scroll-1" || d != image.Pt(0, 90) {
		t.Errorf("node below the viewport: scroll %q by %v, want scroll-1 by (0,90)", c.ID, d)
	}
	// Inside the inner area's bounds, but the inner area is scrolled out
	// of the outer one: the outer one moves.
	if c, d, ok := revealStep(nodes, nested, win); !ok || c.ID != "scroll-1" || d != image.Pt(0, 280) {
		t.Errorf("node in an inner area out of view: scroll %q by %v, want scroll-1 by (0,280)", c.ID, d)
	}
	// Outside the inner area's bounds: the inner one moves first.
	if c, d, ok := revealStep(nodes, deep, win); !ok || c.ID != "scroll-2" || d != image.Pt(0, 240) {
		t.Errorf("node past the inner area: scroll %q by %v, want scroll-2 by (0,240)", c.ID, d)
	}
	// Taller than the viewport: its top goes to the top.
	tall := elements.SemanticNode{ID: "t", Bounds: image.Rect(0, 650, 100, 1450), Scroll: "scroll-1"}
	if _, d, _ := revealStep(append(nodes, tall), tall, win); d != image.Pt(0, 550) {
		t.Errorf("tall node: scroll by %v, want (0,550)", d)
	}
	// Not in any scroll area and off the window: nothing can bring it in.
	off := elements.SemanticNode{ID: "o", Bounds: image.Rect(0, 900, 10, 910)}
	if c, _, ok := revealStep(nodes, off, win); !ok || c.ID != "" {
		t.Errorf("unreachable node: scroll %q, want none", c.ID)
	}
}

func TestScrollArea(t *testing.T) {
	nodes := []elements.SemanticNode{
		{ID: "x", Scroll: "scroll-2"},
		{ID: "scroll-2", Kind: elements.SemanticScroll, Bounds: image.Rect(0, 0, 100, 100), Scroll: "scroll-1"},
		{ID: "scroll-1", Kind: elements.SemanticScroll, Bounds: image.Rect(0, 0, 400, 300)},
		{ID: "scroll-3", Kind: elements.SemanticScroll, Bounds: image.Rect(0, 300, 400, 400)},
		{ID: "plain"},
	}
	for id, want := range map[string]string{"": "scroll-1", "x": "scroll-2", "scroll-3": "scroll-3"} {
		if c, err := scrollArea(nodes, id); err != nil || c.ID != want {
			t.Errorf("scrollArea(%q) = %q, %v; want %q", id, c.ID, err, want)
		}
	}
	if _, err := scrollArea(nodes, "plain"); err == nil {
		t.Error("a node outside any scroll area has no scroll area")
	}
}

func TestFindNode(t *testing.T) {
	nodes := []elements.SemanticNode{
		{ID: "scroll-1", Kind: elements.SemanticScroll, Label: "Messages"},
		{ID: "text@3", Kind: elements.SemanticText, Label: "Old messages"},
		{ID: "button-2", Kind: elements.SemanticButton, Label: "Send"},
	}
	if n, ok := findNode(nodes, "button-2"); !ok || n.Label != "Send" {
		t.Error("by id")
	}
	if n, ok := findNode(nodes, "MESSAGES"); !ok || n.ID != "text@3" {
		t.Errorf("by text found %q, want text@3 (scroll areas skipped)", n.ID)
	}
	exact := []elements.SemanticNode{{ID: "a", Label: "Row 30"}, {ID: "b", Label: "Row 3"}}
	if n, _ := findNode(exact, "row 3"); n.ID != "b" {
		t.Errorf("found %q, want the exact match b", n.ID)
	}
	if _, ok := findNode(nodes, "nothing"); ok {
		t.Error("found a node for unmatched text")
	}
}

func TestFrameStats(t *testing.T) {
	var r frameRecorder
	t0 := time.Unix(0, 0)
	for i := 0; i < 300; i++ {
		start := t0.Add(time.Duration(i) * 10 * time.Millisecond)
		if i >= 150 {
			start = start.Add(time.Second) // an idle gap at frame 150
		}
		build := 2 * time.Millisecond
		if i == 299 {
			build = 30 * time.Millisecond
		}
		r.add(start, start.Add(build), start.Add(build+5*time.Millisecond))
	}
	st := r.stats()
	if st.Frames != 240 {
		t.Errorf("frames %d, want the last 240", st.Frames)
	}
	if st.Build.Max != 30*time.Millisecond || st.Build.P95 != 2*time.Millisecond {
		t.Errorf("build %v, want max 30ms, p95 2ms", st.Build)
	}
	if st.Render.Avg != 5*time.Millisecond {
		t.Errorf("render avg %v, want 5ms", st.Render.Avg)
	}
	if st.Interval.Max != 10*time.Millisecond || int(st.FPS+0.5) != 100 {
		t.Errorf("interval %v, fps %v: want the idle gap left out, 100 fps", st.Interval, st.FPS)
	}
	r.reset()
	if st := r.stats(); st.Frames != 0 || st.FPS != 0 {
		t.Errorf("after reset: %+v", st)
	}
}
