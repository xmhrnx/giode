package scroll

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"

	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

func tallContent(height int) func() elements.Element {
	return func() elements.Element {
		return elements.Box(styles.Styles{Width: -1, Height: properties.Height(height)})
	}
}

func TestScroll_ShortContentIsAtBottomImmediately(t *testing.T) {
	s := New(tallContent(50))

	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: layout.Exact(image.Pt(200, 200))}
	s.Layout(gtx)

	if !s.AtBottom() {
		t.Error("content shorter than the viewport: AtBottom() = false, want true (nothing to scroll)")
	}
	if got := s.Progress(); got != 0 {
		t.Errorf("Progress() = %v, want 0", got)
	}
	if s.ContentHeight() != 50 {
		t.Errorf("ContentHeight() = %d, want 50", s.ContentHeight())
	}
	if s.ViewportHeight() != 200 {
		t.Errorf("ViewportHeight() = %d, want 200", s.ViewportHeight())
	}
}

func TestScroll_OnReachBottomFiresOnceOnRisingEdge(t *testing.T) {
	s := New(tallContent(1000))

	calls := 0
	s.OnReachBottom(func() { calls++ })

	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: layout.Exact(image.Pt(200, 200))}

	s.Layout(gtx) // not at bottom yet (content 1000 > viewport 200)
	if calls != 0 {
		t.Fatalf("calls = %d before reaching bottom, want 0", calls)
	}

	s.ScrollTo(800) // maxOffset = 1000-200 = 800
	s.Layout(gtx)
	if calls != 1 {
		t.Fatalf("calls = %d after scrolling to bottom, want 1", calls)
	}

	// Staying at the bottom across more frames doesn't fire again.
	s.Layout(gtx)
	s.Layout(gtx)
	if calls != 1 {
		t.Fatalf("calls = %d after staying at the bottom, want 1 (edge-triggered)", calls)
	}

	// Scrolling away and back fires it again.
	s.ScrollTo(0)
	s.Layout(gtx)
	if calls != 1 {
		t.Fatalf("calls = %d after scrolling away from the bottom, want 1 (unchanged)", calls)
	}
	s.ScrollTo(800)
	s.Layout(gtx)
	if calls != 2 {
		t.Fatalf("calls = %d after scrolling back to the bottom, want 2", calls)
	}
}

func TestScroll_Progress(t *testing.T) {
	s := New(tallContent(1000))

	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: layout.Exact(image.Pt(200, 200))}
	s.Layout(gtx) // maxOffset = 800

	s.ScrollTo(400)
	s.Layout(gtx)
	if got, want := s.Progress(), float32(0.5); got != want {
		t.Errorf("Progress() at offset 400/800 = %v, want %v", got, want)
	}

	s.ScrollTo(800)
	s.Layout(gtx)
	if got, want := s.Progress(), float32(1); got != want {
		t.Errorf("Progress() at offset 800/800 = %v, want %v", got, want)
	}
}

func TestScroll_Visible(t *testing.T) {
	s := New(tallContent(1000))

	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: layout.Exact(image.Pt(200, 200))}
	s.ScrollTo(300)
	s.Layout(gtx) // viewport now covers content [300, 500)

	cases := []struct {
		top, height int
		want        bool
	}{
		{0, 100, false},   // [0,100) entirely above the viewport
		{600, 100, false}, // [600,700) entirely below the viewport
		{250, 100, true},  // [250,350) overlaps the top edge
		{450, 100, true},  // [450,550) overlaps the bottom edge
		{320, 50, true},   // [320,370) fully inside
		{300, 200, true},  // exactly the viewport
	}
	for _, c := range cases {
		if got := s.Visible(c.top, c.height); got != c.want {
			t.Errorf("Visible(%d, %d) = %v, want %v", c.top, c.height, got, c.want)
		}
	}
}

func TestScroll_TrackedPositionsFollowOffset(t *testing.T) {
	var pos image.Point
	probe := elements.Raw(func(gtx layout.Context) layout.Dimensions {
		elements.TrackPosition(&pos)
		return layout.Dimensions{Size: image.Pt(10, 10)}
	})
	s := New(func() elements.Element {
		return elements.Stack(styles.Styles{}, elements.Box(styles.Styles{Height: 300}), probe, elements.Box(styles.Styles{Height: 300}))
	})
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: layout.Exact(image.Pt(200, 200))}
	s.Layout(gtx)
	s.ScrollTo(100)
	ops.Reset()
	s.Layout(gtx)
	if want := image.Pt(0, 300-int(s.Offset())); s.Offset() == 0 || pos != want {
		t.Errorf("tracked position %v (offset %v), want %v", pos, s.Offset(), want)
	}
}

func layoutScroll(s *Scroll, max image.Point) layout.Dimensions {
	var ops op.Ops
	return s.Layout(layout.Context{Ops: &ops, Constraints: layout.Constraints{Max: max}})
}

func TestScroll_AnchorBottomFollowsUntilScrolledUp(t *testing.T) {
	h := 300
	s := New(func() elements.Element { return elements.Box(styles.Styles{Width: -1, Height: properties.Height(h)}) }).Anchor(AnchorBottom)
	layoutScroll(s, image.Pt(200, 200))
	if s.Offset() != 100 {
		t.Fatalf("starts at offset %v, want the bottom (100)", s.Offset())
	}
	h = 500 // a reply streams in
	layoutScroll(s, image.Pt(200, 200))
	if s.Offset() != 300 {
		t.Fatalf("offset %v after growth, want to follow to 300", s.Offset())
	}
	// The user scrolls up with the wheel: growth no longer moves the
	// offset.
	var r input.Router
	frame := func() {
		var ops op.Ops
		s.Layout(layout.Context{Ops: &ops, Source: r.Source(), Constraints: layout.Constraints{Max: image.Pt(200, 200)}})
		r.Frame(&ops)
	}
	frame()
	r.Queue(pointer.Event{Kind: pointer.Scroll, Source: pointer.Mouse, Position: f32.Pt(50, 50), Scroll: f32.Pt(0, -250)})
	frame()
	if s.Offset() != 50 {
		t.Fatalf("offset %v after scrolling up 250, want 50", s.Offset())
	}
	h = 700
	frame()
	if s.Offset() != 50 {
		t.Errorf("offset %v after more content, want it kept at 50", s.Offset())
	}
	s.ScrollToBottom()
	h = 900
	layoutScroll(s, image.Pt(200, 200))
	if s.Offset() != 700 {
		t.Errorf("offset %v after ScrollToBottom, want 700", s.Offset())
	}
}

func TestScroll_FitContent(t *testing.T) {
	h := 90
	s := New(func() elements.Element { return elements.Box(styles.Styles{Width: -1, Height: properties.Height(h)}) }).FitContent(true)
	if got := layoutScroll(s, image.Pt(200, 400)).Size.Y; got != 90 {
		t.Errorf("fit viewport %d, want the content's 90", got)
	}
	h = 900
	if got := layoutScroll(s, image.Pt(200, 400)).Size.Y; got != 400 {
		t.Errorf("fit viewport %d, want capped at 400", got)
	}
}

func TestScroll_Horizontal(t *testing.T) {
	var seen []float32
	s := New(func() elements.Element { return elements.Box(styles.Styles{Width: 1000, Height: 30}) }).
		Axis(Horizontal).OnScroll(func(off, max float32) { seen = append(seen, off, max) })
	dims := layoutScroll(s, image.Pt(300, 1e6))
	if dims.Size != image.Pt(300, 30) {
		t.Fatalf("viewport %v, want 300x30 (as tall as the content)", dims.Size)
	}
	s.ScrollToX(5000)
	layoutScroll(s, image.Pt(300, 1e6))
	if s.OffsetX() != 700 {
		t.Errorf("x offset %v, want clamped to 700", s.OffsetX())
	}
	if len(seen) < 2 || seen[len(seen)-2] != 700 || seen[len(seen)-1] != 700 {
		t.Errorf("OnScroll calls %v, want the last to report 700 of 700", seen)
	}
}

func TestScroll_FingerScrollsUpWhileContentGrows(t *testing.T) {
	h := 400
	s := New(func() elements.Element { return elements.Box(styles.Styles{Width: -1, Height: properties.Height(h)}) }).Anchor(AnchorBottom)
	var r input.Router
	frame := func() {
		h += 30 // a reply streams in
		var ops op.Ops
		s.Layout(layout.Context{Ops: &ops, Source: r.Source(), Constraints: layout.Constraints{Max: image.Pt(200, 200)}})
		r.Frame(&ops)
	}
	touch := func(kind pointer.Kind, y float32) {
		r.Queue(pointer.Event{Kind: kind, Source: pointer.Touch, Position: f32.Pt(50, y), PointerID: 1})
		frame()
	}
	frame()
	frame()
	// A little towards the end first, then down: scrolling up.
	touch(pointer.Press, 100)
	touch(pointer.Move, 90)
	touch(pointer.Move, 85)
	touch(pointer.Move, 150)
	off := s.Offset()
	for i := 0; i < 5; i++ {
		frame()
		if s.Offset() != off {
			t.Fatalf("frame %d: offset %v while the finger rests, want %v", i, s.Offset(), off)
		}
	}
	touch(pointer.Release, 150)
	frame()
	if s.Offset() != off {
		t.Errorf("offset %v after letting go above the bottom, want %v", s.Offset(), off)
	}
}
