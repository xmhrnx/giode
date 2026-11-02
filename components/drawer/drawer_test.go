package drawer

import (
	"image"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/mehran9675/giode/elements"
	"github.com/mehran9675/giode/styles"
)

type harness struct {
	r    input.Router
	d    *Drawer
	now  time.Time
	dims layout.Dimensions
}

func (h *harness) frame() {
	h.now = h.now.Add(50 * time.Millisecond)
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Now: h.now, Source: h.r.Source(), Constraints: layout.Constraints{Max: image.Pt(800, 600)}}
	h.dims = h.d.Layout(gtx)
	elements.DrainPortals(gtx)
	h.r.Frame(&ops)
}

func (h *harness) settle() {
	for i := 0; i < 10; i++ {
		h.frame()
	}
}

func (h *harness) drag(from, to float32) {
	h.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Touch, Position: f32.Pt(from, 300), Buttons: pointer.ButtonPrimary})
	h.frame()
	for i := 1; i <= 5; i++ {
		x := from + (to-from)*float32(i)/5
		h.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Touch, Position: f32.Pt(x, 300), Buttons: pointer.ButtonPrimary})
		h.frame()
	}
	h.r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Touch, Position: f32.Pt(to, 300)})
	h.frame()
}

func newHarness(open *bool) *harness {
	d := New(open, Left, 200, func() elements.Element { return elements.Box(styles.Styles{}) })
	return &harness{d: d, now: time.Unix(0, 0)}
}

func TestDrawer_TakesNoSpace(t *testing.T) {
	open := new(bool)
	h := newHarness(open)
	h.frame()
	*open = true
	h.settle()
	if h.dims.Size != (image.Point{}) {
		t.Errorf("open drawer took %v of the layout, want none", h.dims.Size)
	}
	if h.d.progress != 1 {
		t.Errorf("progress %v after settling, want 1", h.d.progress)
	}
}

func TestDrawer_DragAndScrimClose(t *testing.T) {
	open := new(bool)
	h := newHarness(open)
	*open = true
	h.settle()

	h.drag(150, 10) // pull the panel back toward the left edge
	if *open {
		t.Fatal("dragging the panel closed didn't close it")
	}
	h.settle()

	*open = true
	h.settle()
	h.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Touch, Position: f32.Pt(600, 300), Buttons: pointer.ButtonPrimary},
		pointer.Event{Kind: pointer.Release, Source: pointer.Touch, Position: f32.Pt(600, 300)})
	h.frame()
	h.frame()
	if *open {
		t.Error("tapping the scrim didn't close the drawer")
	}
}

func TestDrawer_SwipeToOpen(t *testing.T) {
	open := new(bool)
	h := newHarness(open)
	h.frame()
	h.drag(5, 180)
	if *open {
		t.Fatal("edge swipe opened a drawer without SwipeToOpen")
	}
	h.d.SwipeToOpen(true)
	h.frame()
	h.drag(5, 180)
	if !*open {
		t.Error("edge swipe didn't open the drawer")
	}
}

func TestNotAnimated(t *testing.T) {
	open := false
	for _, d := range []*Drawer{
		New(&open, Left, 200, func() elements.Element { return elements.Box(styles.Styles{}) }).Animated(false),
		New(&open, Left, 200, func() elements.Element { return elements.Box(styles.Styles{}) }).Duration(0),
	} {
		h := &harness{d: d}
		h.frame()
		open = true
		h.frame()
		if d.progress != 1 {
			t.Errorf("one frame after opening: progress %v, want 1", d.progress)
		}
		open = false
		h.frame()
		if d.progress != 0 {
			t.Errorf("one frame after closing: progress %v, want 0", d.progress)
		}
	}
	// The default still slides.
	d := New(&open, Left, 200, func() elements.Element { return elements.Box(styles.Styles{}) })
	h := &harness{d: d}
	h.frame()
	open = true
	h.frame()
	if d.progress <= 0 || d.progress >= 1 {
		t.Errorf("the default drawer jumped to %v in one frame", d.progress)
	}
	if d.Animated(false).Animated(true).duration != animDuration {
		t.Error("Animated(true) didn't restore the default duration")
	}
}
