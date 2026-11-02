package swipeable

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
	"github.com/xmhrnx/giode/styles"
)

type harness struct {
	r   input.Router
	s   *Swipeable
	now time.Time
}

func (h *harness) frame() {
	h.now = h.now.Add(30 * time.Millisecond)
	var ops op.Ops
	h.s.Layout(layout.Context{Ops: &ops, Now: h.now, Source: h.r.Source(), Constraints: layout.Exact(image.Pt(400, 60))})
	h.r.Frame(&ops)
}

func (h *harness) settle() {
	for i := 0; i < 12; i++ {
		h.frame()
	}
}

func (h *harness) drag(from, to f32.Point) {
	h.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Touch, Position: from, Buttons: pointer.ButtonPrimary})
	h.frame()
	for i := 1; i <= 5; i++ {
		p := from.Add(to.Sub(from).Mul(float32(i) / 5))
		h.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Touch, Position: p, Buttons: pointer.ButtonPrimary})
		h.frame()
	}
	h.r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Touch, Position: to})
	h.frame()
	h.settle()
}

func (h *harness) tap(x, y float32) {
	p := f32.Pt(x, y)
	h.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Touch, Position: p, Buttons: pointer.ButtonPrimary},
		pointer.Event{Kind: pointer.Release, Source: pointer.Touch, Position: p})
	h.frame()
	h.settle()
}

func TestSwipeable(t *testing.T) {
	deleted, rowTaps := 0, 0
	row := elements.NewPressable(func(elements.PressState) elements.Element {
		return elements.Box(styles.Styles{Width: -1, Height: 60, Background: "#ffffff"})
	}).OnClick(func(elements.ClickEvent) { rowTaps++ })
	s := New(func() elements.Element { return row }, nil, []Action{{Label: "Delete", OnTap: func() { deleted++ }}})
	h := &harness{s: s, now: time.Unix(0, 0)}
	h.settle()

	h.drag(f32.Pt(300, 30), f32.Pt(290, 30))
	if s.Opened() {
		t.Fatal("a short swipe opened the row")
	}
	h.drag(f32.Pt(300, 30), f32.Pt(300, 90)) // ends off the row, so no click
	if s.Opened() {
		t.Fatal("a vertical drag opened the row")
	}
	if rowTaps != 0 {
		t.Fatalf("swipes counted as %d row taps", rowTaps)
	}
	h.drag(f32.Pt(300, 30), f32.Pt(150, 30))
	if !s.Opened() || s.offset != -80 {
		t.Fatalf("swipe left: opened=%v offset=%v, want open at -80", s.Opened(), s.offset)
	}
	h.tap(380, 30) // the revealed Delete button
	if deleted != 1 || s.Opened() {
		t.Fatalf("tapping Delete: deleted=%d opened=%v", deleted, s.Opened())
	}
	h.drag(f32.Pt(300, 30), f32.Pt(150, 30))
	h.tap(100, 30) // the row itself closes it, without a click
	if s.Opened() || rowTaps != 0 {
		t.Errorf("tap on open row: opened=%v rowTaps=%d, want closed and no click", s.Opened(), rowTaps)
	}
	h.tap(100, 30)
	if rowTaps != 1 {
		t.Errorf("tap on closed row: rowTaps=%d, want 1", rowTaps)
	}
}

func TestSwipeable_SetActions(t *testing.T) {
	var got []string
	pin := func(label string) []Action {
		return []Action{{Label: label, OnTap: func() { got = append(got, label) }}}
	}
	row := elements.Box(styles.Styles{Width: -1, Height: 60, Background: "#ffffff"})
	s := New(func() elements.Element { return row }, nil, pin("Pin"))
	h := &harness{s: s, now: time.Unix(0, 0)}
	h.settle()
	h.drag(f32.Pt(300, 30), f32.Pt(150, 30))
	h.tap(380, 30)
	s.SetActions(nil, pin("Unpin"))
	h.drag(f32.Pt(300, 30), f32.Pt(150, 30))
	h.tap(380, 30)
	if len(got) != 2 || got[0] != "Pin" || got[1] != "Unpin" {
		t.Fatalf("taps ran %v, want [Pin Unpin]", got)
	}

	// Two trailing actions open wider; dropping back to one, or to
	// none, resizes or closes the open row.
	s.SetActions(nil, append(pin("A"), pin("B")...))
	h.drag(f32.Pt(390, 30), f32.Pt(100, 30))
	h.settle()
	if s.offset != -160 {
		t.Fatalf("open with two actions at %v, want -160", s.offset)
	}
	s.SetActions(nil, pin("A"))
	h.settle()
	if s.offset != -80 {
		t.Errorf("after dropping to one action, offset %v, want -80", s.offset)
	}
	s.SetActions(nil, nil)
	h.settle()
	if s.Opened() || s.offset != 0 {
		t.Errorf("with no actions left: opened=%v offset=%v, want closed", s.Opened(), s.offset)
	}
}
