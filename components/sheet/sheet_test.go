package sheet

import (
	"image"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/styles"
)

type harness struct {
	r     input.Router
	s     *Sheet
	now   time.Time
	win   image.Point
	nodes []elements.SemanticNode
}

func (h *harness) frame() {
	h.now = h.now.Add(40 * time.Millisecond)
	elements.SetWindowSize(h.win)
	elements.BeginFrame()
	elements.BeginSemantics()
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Now: h.now, Source: h.r.Source(), Constraints: layout.Constraints{Max: h.win}}
	h.s.Layout(gtx)
	elements.DrainPortals(gtx)
	elements.DispatchBack(gtx)
	elements.RunAfterLayout()
	h.nodes = elements.EndSemantics()
	h.r.Frame(&ops)
}

func (h *harness) settle() {
	for i := 0; i < 12; i++ {
		h.frame()
	}
}

func (h *harness) node(label string) (elements.SemanticNode, bool) {
	for _, n := range h.nodes {
		if n.Label == label {
			return n, true
		}
	}
	return elements.SemanticNode{}, false
}

func newHarness(win image.Point) (*harness, *bool) {
	fonts.UseGofont()
	open := new(bool)
	s := New(open, func() elements.Element {
		return elements.Stack(styles.Styles{}, elements.Box(styles.Styles{Height: 100}, elements.Text("Body")))
	}).Title("Pick one")
	return &harness{s: s, now: time.Unix(0, 0), win: win}, open
}

func TestSheet_BottomFitsContent(t *testing.T) {
	h, open := newHarness(image.Pt(400, 800))
	h.frame()
	*open = true
	h.settle()
	body, ok := h.node("Body")
	if !ok {
		t.Fatal("body not shown")
	}
	if body.Bounds.Min.Y < 600 || body.Bounds.Min.Y > 760 {
		t.Errorf("body at y=%d, want near the bottom of an 800px window", body.Bounds.Min.Y)
	}
	if _, ok := h.node("Pick one"); !ok {
		t.Error("title not shown")
	}
	// The scrim, above the sheet, closes it.
	h.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Touch, Position: f32.Pt(200, 100), Buttons: pointer.ButtonPrimary},
		pointer.Event{Kind: pointer.Release, Source: pointer.Touch, Position: f32.Pt(200, 100)})
	h.frame()
	h.frame()
	if *open {
		t.Error("scrim tap didn't close the sheet")
	}
}

func TestSheet_DragHandleDown(t *testing.T) {
	h, open := newHarness(image.Pt(400, 800))
	h.frame()
	*open = true
	h.settle()
	title, _ := h.node("Pick one")
	from := f32.Pt(200, float32(title.Bounds.Min.Y))
	h.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Touch, Position: from, Buttons: pointer.ButtonPrimary})
	h.frame()
	for i := 1; i <= 5; i++ {
		h.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Touch, Position: from.Add(f32.Pt(0, float32(i*30))), Buttons: pointer.ButtonPrimary})
		h.frame()
	}
	h.r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Touch, Position: from.Add(f32.Pt(0, 150))})
	h.frame()
	if *open {
		t.Error("dragging the handle down didn't close the sheet")
	}
}

func TestSheet_DetentsAndAdaptive(t *testing.T) {
	h, open := newHarness(image.Pt(400, 800))
	h.s.Detents(DetentMedium, DetentLarge)
	h.frame()
	*open = true
	h.settle()
	title, _ := h.node("Pick one")
	if y := title.Bounds.Min.Y; y < 380 || y > 440 {
		t.Errorf("medium sheet title at y=%d, want about halfway down", y)
	}

	w, open2 := newHarness(image.Pt(1000, 800))
	w.s.Adaptive(true)
	w.frame()
	*open2 = true
	w.settle()
	body, _ := w.node("Body")
	if cx := (body.Bounds.Min.Y + body.Bounds.Max.Y) / 2; cx < 300 || cx > 500 {
		t.Errorf("adaptive dialog body centered at y=%d, want near the middle", cx)
	}
	_ = open
}
