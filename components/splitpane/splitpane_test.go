package splitpane

import (
	"image"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/elements"
)

type harness struct {
	r    input.Router
	s    *SplitPane
	size image.Point
	now  time.Time
	a, b image.Point // the panes' sizes
}

func newHarness(s *SplitPane, size image.Point) *harness {
	h := &harness{s: s, size: size, now: time.Unix(0, 0)}
	s.Content(h.pane(&h.a), h.pane(&h.b))
	h.frame()
	return h
}

func (h *harness) pane(dst *image.Point) func() elements.Element {
	return func() elements.Element {
		return elements.Raw(func(gtx layout.Context) layout.Dimensions {
			*dst = gtx.Constraints.Max
			return layout.Dimensions{Size: gtx.Constraints.Max}
		})
	}
}

func (h *harness) frame() {
	h.now = h.now.Add(time.Second)
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Now: h.now, Source: h.r.Source(), Constraints: layout.Exact(h.size)}
	h.s.Layout(gtx)
	h.r.Frame(&ops)
}

func (h *harness) drag(from, to f32.Point) {
	h.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: from})
	h.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: from, Buttons: pointer.ButtonPrimary})
	h.frame()
	h.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: to, Buttons: pointer.ButtonPrimary})
	h.frame()
	h.r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: to})
	h.frame()
	h.frame()
}

func TestSplitPaneDrag(t *testing.T) {
	var reported float32
	s := New(nil, nil).Ratio(0.25).OnResize(func(r, dp float32) { reported = dp })
	h := newHarness(s, image.Pt(401, 200))
	if h.a.X != 100 || h.b.X != 300 || h.a.Y != 200 {
		t.Fatalf("panes %v and %v, want 100 and 300 wide", h.a, h.b)
	}
	h.drag(f32.Pt(100, 50), f32.Pt(160, 50))
	if h.a.X != 160 || h.b.X != 240 || reported != 160 {
		t.Errorf("after dragging 60px: panes %v and %v, reported %v", h.a, h.b, reported)
	}
	// The share holds as the space grows.
	h.size = image.Pt(801, 200)
	h.frame()
	if h.a.X != 320 {
		t.Errorf("at twice the width: first pane %d, want 320", h.a.X)
	}
	// Not below the second pane's minimum.
	h.drag(f32.Pt(320, 50), f32.Pt(790, 50))
	if h.b.X != 48 {
		t.Errorf("dragged to the end: second pane %d, want its 48 minimum", h.b.X)
	}
}

func TestSplitPaneFixedAndKeys(t *testing.T) {
	s := New(nil, nil).FirstSize(120).Axis(Vertical)
	h := newHarness(s, image.Pt(300, 401))
	if h.a.Y != 120 || h.b.Y != 280 {
		t.Fatalf("panes %v and %v", h.a, h.b)
	}
	h.size = image.Pt(300, 601)
	h.frame()
	if h.a.Y != 120 {
		t.Errorf("a fixed first pane changed to %d", h.a.Y)
	}
	h.r.Queue(key.FocusEvent{Focus: true})
	h.frame()
	// Focus it, then move it down with the arrow key.
	h.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: f32.Pt(10, 120), Buttons: pointer.ButtonPrimary},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: f32.Pt(10, 120)})
	h.frame()
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Source: h.r.Source()}
	gtx.Execute(key.FocusCmd{Tag: s})
	h.frame()
	h.r.Queue(key.Event{Name: key.NameDownArrow, State: key.Press})
	h.frame()
	h.frame()
	if h.a.Y != 136 {
		t.Errorf("after Down: first pane %d, want 136", h.a.Y)
	}
}
