package elements

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/styles"
)

// memoHarness lays out a memo over a router, frame by frame.
type memoHarness struct {
	r      input.Router
	c      MemoCache
	build  func() Element
	builds int
}

func (h *memoHarness) frame() {
	BeginFrame()
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Source: h.r.Source(), Constraints: layout.Exact(image.Pt(400, 200)), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
	h.c.Layout(gtx, func() Element { h.builds++; return h.build() })
	h.r.Frame(&ops)
	EndFrame()
}

func (h *memoHarness) move(x, y float32) {
	h.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: f32.Pt(x, y)})
	h.frame()
}

func TestMemoCache_InputWithoutRebuild(t *testing.T) {
	clicks := 0
	btn := NewPressable(func(s PressState) Element {
		bg := "#dddddd"
		if s.Hovered {
			bg = "#bbbbbb"
		}
		return Box(styles.Styles{Width: 100, Height: 40, Background: bg})
	}).OnClick(func(ClickEvent) { clicks++ })
	h := &memoHarness{build: func() Element {
		return Stack(styles.Styles{Width: -1, Height: -1},
			btn,
			Text("Some static text in the cached block", styles.Styles{}),
		)
	}}
	// The first frames settle (Gio sends new handlers a reset event).
	for i := 0; i < 3; i++ {
		h.frame()
	}
	base := h.builds
	h.frame()
	h.frame()
	if h.builds != base {
		t.Fatalf("idle frames rebuilt %d times", h.builds-base)
	}

	// Moving over static content: no rebuild.
	for i := 0; i < 5; i++ {
		h.move(200+float32(i), 10)
	}
	if h.builds != base {
		t.Errorf("pointer moves over static content rebuilt %d times", h.builds-base)
	}

	// Entering the button (100x40 at the top left) changes its hover
	// state: one rebuild.
	var btnY float32 = 20
	h.move(50, btnY)
	h.frame()
	if h.builds != base+1 {
		t.Fatalf("entering the button: %d rebuilds, want 1", h.builds-base)
	}
	if !btn.State().Hovered {
		t.Fatal("button didn't see the hover")
	}
	// Moving within it: no more rebuilds.
	for i := 0; i < 5; i++ {
		h.move(50+float32(i), btnY)
	}
	if h.builds != base+1 {
		t.Errorf("moving within the button rebuilt %d more times", h.builds-base-1)
	}

	// A click reaches the button exactly once.
	h.r.Queue(
		pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: f32.Pt(55, btnY), Buttons: pointer.ButtonPrimary},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: f32.Pt(55, btnY)},
	)
	h.frame()
	h.frame()
	if clicks != 1 {
		t.Errorf("clicks = %d, want 1", clicks)
	}
	// Idle frames after the click replay.
	n := h.builds
	for i := 0; i < 3; i++ {
		h.frame()
	}
	if h.builds != n {
		t.Errorf("idle frames after the click rebuilt %d times", h.builds-n)
	}
}

func TestMemoCache_KeepsKeyedStateAlive(t *testing.T) {
	made := 0
	h := &memoHarness{build: func() Element {
		return Keyed("memo-test", 1, func() Element {
			made++
			return Box(styles.Styles{Width: 10, Height: 10})
		})
	}}
	for i := 0; i < DefaultRetainFrames*3; i++ {
		h.frame()
	}
	if h.builds > 2 {
		t.Fatalf("built %d times", h.builds)
	}
	h.c.Invalidate()
	h.frame()
	if made != 1 {
		t.Errorf("keyed state recreated after %d replayed frames", DefaultRetainFrames*3)
	}
}

func TestMemo_NestedPassesReachInnerWidgets(t *testing.T) {
	clicks := 0
	btn := NewPressable(func(PressState) Element { return Box(styles.Styles{Width: 100, Height: 40}) }).
		OnClick(func(ClickEvent) { clicks++ })
	inner := NewMemoCache()
	h := &memoHarness{build: func() Element {
		return Raw(func(gtx layout.Context) layout.Dimensions {
			return inner.Layout(gtx, func() Element { return btn })
		})
	}}
	h.frame()
	h.frame()
	h.r.Queue(
		pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: f32.Pt(20, 20), Buttons: pointer.ButtonPrimary},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: f32.Pt(20, 20)},
	)
	h.frame()
	h.frame()
	if clicks != 1 {
		t.Errorf("clicks through nested caches = %d, want 1", clicks)
	}
}
