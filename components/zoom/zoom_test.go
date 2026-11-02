package zoom

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
	"gioui.org/widget"

	"github.com/xmhrnx/giode/elements"
)

type harness struct {
	r    input.Router
	z    *Zoom
	now  time.Time
	view image.Point
}

func newHarness(content func() elements.Element) *harness {
	return &harness{z: New(content), now: time.Unix(0, 0), view: image.Pt(400, 400)}
}

func (h *harness) frame() {
	h.now = h.now.Add(16 * time.Millisecond)
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Now: h.now, Source: h.r.Source(), Constraints: layout.Constraints{Max: h.view}}
	h.z.Layout(gtx)
	h.r.Frame(&ops)
}

// settle runs frames until animations finish.
func (h *harness) settle() {
	for i := 0; i < 30; i++ {
		h.frame()
	}
}

func (h *harness) touch(kind pointer.Kind, id pointer.ID, x, y float32) {
	h.r.Queue(pointer.Event{Kind: kind, Source: pointer.Touch, PointerID: id, Position: f32.Pt(x, y)})
	h.frame()
}

// page is a 200x300 content block, centred in the 400x400 view at
// (100, 50).
func page() elements.Element {
	return elements.Raw(func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: image.Pt(200, 300)}
	})
}

func TestZoom_FitsAndCentres(t *testing.T) {
	h := newHarness(page)
	h.frame()
	if h.z.Scale() != 1 || h.z.Offset() != f32.Pt(100, 50) {
		t.Fatalf("scale %v offset %v, want 1 and (100,50)", h.z.Scale(), h.z.Offset())
	}
}

func TestZoom_PinchZoomsAroundTheFingers(t *testing.T) {
	h := newHarness(page)
	h.frame()
	// Two fingers around (200, 200), 40px apart, spread to 120px.
	h.touch(pointer.Press, 1, 180, 200)
	h.touch(pointer.Press, 2, 220, 200)
	before := h.z.ToContent(f32.Pt(200, 200))
	for i := 1; i <= 4; i++ {
		d := float32(20 + 10*i)
		h.touch(pointer.Move, 1, 200-d, 200)
		h.touch(pointer.Move, 2, 200+d, 200)
	}
	if s := h.z.Scale(); s < 2.9 || s > 3.1 {
		t.Fatalf("scale after spreading 40px to 120px: %v, want 3", s)
	}
	if after := h.z.ToContent(f32.Pt(200, 200)); abs(after.X-before.X) > 0.5 || abs(after.Y-before.Y) > 0.5 {
		t.Errorf("the point under the fingers moved: %v -> %v", before, after)
	}
	h.touch(pointer.Release, 1, 140, 200)
	h.touch(pointer.Release, 2, 260, 200)
	if !h.z.Zoomed() {
		t.Error("not zoomed after the pinch")
	}
}

func TestZoom_OneFingerPansOnlyWhenZoomed(t *testing.T) {
	h := newHarness(page)
	h.frame()
	h.touch(pointer.Press, 1, 200, 200)
	h.touch(pointer.Move, 1, 150, 150)
	h.touch(pointer.Release, 1, 150, 150)
	if h.z.Offset() != f32.Pt(100, 50) {
		t.Fatalf("panned at scale 1: offset %v", h.z.Offset())
	}
	h.z.SetScale(3)
	h.settle()
	off := h.z.Offset()
	h.touch(pointer.Press, 1, 200, 200)
	h.touch(pointer.Move, 1, 180, 170)
	h.touch(pointer.Move, 1, 160, 140)
	h.touch(pointer.Release, 1, 160, 140)
	if got := h.z.Offset(); got.X != off.X-40 || got.Y != off.Y-60 {
		t.Errorf("pan by (-40,-60) moved %v to %v", off, got)
	}
	// It stops at the content's edges.
	h.touch(pointer.Press, 1, 200, 200)
	h.touch(pointer.Move, 1, 2000, 2000)
	h.touch(pointer.Release, 1, 2000, 2000)
	if got := h.z.Offset(); got.X != 0 || got.Y != 0 {
		t.Errorf("panned past the top left edge: %v", got)
	}
}

func TestZoom_DoubleTap(t *testing.T) {
	h := newHarness(page)
	h.frame()
	tap := func() {
		h.touch(pointer.Press, 1, 150, 100)
		h.touch(pointer.Release, 1, 150, 100)
	}
	tap()
	tap()
	h.settle()
	if s := h.z.Scale(); s != 2.5 {
		t.Fatalf("double tap zoomed to %v, want 2.5", s)
	}
	h.now = h.now.Add(time.Second)
	tap()
	tap()
	h.settle()
	if s := h.z.Scale(); s != 1 {
		t.Errorf("second double tap zoomed to %v, want back to 1", s)
	}
}

func TestZoom_ModifierWheelZooms(t *testing.T) {
	h := newHarness(page)
	h.frame()
	h.r.Queue(pointer.Event{Kind: pointer.Scroll, Source: pointer.Mouse, Position: f32.Pt(200, 200), Scroll: f32.Pt(0, -200), Modifiers: key.ModShortcut})
	h.frame()
	// At scale 1 Zoom takes no wheel unless WheelZoom: nothing to pan.
	if h.z.Scale() != 1 {
		t.Fatalf("wheel zoomed a Zoom without WheelZoom at scale 1: %v", h.z.Scale())
	}
	h.z.WheelZoom(true)
	h.frame()
	h.r.Queue(pointer.Event{Kind: pointer.Scroll, Source: pointer.Mouse, Position: f32.Pt(200, 200), Scroll: f32.Pt(0, -200)})
	h.frame()
	if !h.z.Zoomed() {
		t.Errorf("WheelZoom: scrolling up didn't zoom in (scale %v)", h.z.Scale())
	}
}

func TestZoom_ContentKeepsItsOwnTaps(t *testing.T) {
	var btn widget.Clickable
	clicks := 0
	h := newHarness(func() elements.Element {
		return elements.Raw(func(gtx layout.Context) layout.Dimensions {
			if btn.Clicked(gtx) {
				clicks++
			}
			return btn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Dimensions{Size: image.Pt(200, 300)}
			})
		})
	})
	h.frame()
	h.z.SetScale(2)
	h.settle()
	// The content's centre is at the view's centre.
	h.touch(pointer.Press, 1, 200, 200)
	h.touch(pointer.Release, 1, 200, 200)
	h.frame()
	if clicks != 1 {
		t.Errorf("a tap on a button inside the zoomed content: %d clicks, want 1", clicks)
	}
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
