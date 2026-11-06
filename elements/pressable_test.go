package elements

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

	"github.com/xmhrnx/giode/styles"
)

// harness lays out an element against a real Gio router, so tests can
// queue pointer and key events.
type harness struct {
	r   input.Router
	el  Element
	now time.Time
}

func (h *harness) frame() {
	var ops op.Ops
	h.now = h.now.Add(16 * time.Millisecond)
	gtx := layout.Context{Ops: &ops, Now: h.now, Constraints: layout.Constraints{Max: image.Pt(400, 400)}, Source: h.r.Source()}
	h.el.Layout(gtx)
	h.r.Frame(&ops)
}

func (h *harness) click(x, y float32, btn pointer.Buttons, mods key.Modifiers) {
	pos := f32.Pt(x, y)
	h.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: pos})
	h.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: pos, Buttons: btn, Modifiers: mods})
	h.frame()
	h.r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: pos, Modifiers: mods})
	h.frame()
}

func sized(w, h int) Element {
	return Raw(func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: image.Pt(w, h)}
	})
}

func TestPressable_ClickKinds(t *testing.T) {
	var events []ClickEvent
	var doubles, rights int
	p := NewPressable(func(PressState) Element { return sized(100, 40) }).
		OnClick(func(e ClickEvent) { events = append(events, e) }).
		OnDoubleClick(func(ClickEvent) { doubles++ }).
		OnSecondaryClick(func(ClickEvent) { rights++ })
	h := &harness{el: p, now: time.Unix(0, 0)}
	h.frame()

	h.click(10, 10, pointer.ButtonPrimary, key.ModShift)
	h.click(10, 10, pointer.ButtonPrimary, 0)
	if len(events) != 2 || events[0].Clicks != 1 || events[1].Clicks != 2 || doubles != 1 {
		t.Fatalf("events %+v, doubles %d: want a single then a double click", events, doubles)
	}
	if !events[0].Shift() || events[0].Source != ClickPointer || events[0].Position != image.Pt(10, 10) {
		t.Errorf("first event = %+v", events[0])
	}
	h.now = h.now.Add(time.Second)
	h.click(10, 10, pointer.ButtonSecondary, 0)
	if rights != 1 || events[2].Button != MouseSecondary || events[2].Clicks != 1 {
		t.Errorf("right click: %d calls, event %+v", rights, events[2])
	}
	h.click(300, 300, pointer.ButtonPrimary, 0)
	if len(events) != 3 {
		t.Error("click outside the content fired")
	}
}

func TestPressable_ChildrenKeepTheirClicks(t *testing.T) {
	var outer, inner int
	child := NewPressable(func(PressState) Element { return sized(20, 20) }).OnClick(func(ClickEvent) { inner++ })
	p := NewPressable(func(PressState) Element {
		return Row(styles.Styles{}, child, sized(80, 20))
	}).OnClick(func(ClickEvent) { outer++ })
	h := &harness{el: p, now: time.Unix(0, 0)}
	h.frame()
	h.click(5, 5, pointer.ButtonPrimary, 0)
	if inner != 1 || outer != 0 {
		t.Errorf("click on child: inner %d, outer %d - want the child only", inner, outer)
	}
	h.now = h.now.Add(time.Second)
	h.click(50, 5, pointer.ButtonPrimary, 0)
	if outer != 1 {
		t.Errorf("click beside the child: outer %d, want 1", outer)
	}
}

func TestPressable_KeyboardAndState(t *testing.T) {
	var events []ClickEvent
	var states []PressState
	p := NewPressable(func(s PressState) Element {
		states = append(states, s)
		return sized(100, 40)
	}).OnClick(func(e ClickEvent) { events = append(events, e) })
	h := &harness{el: p, now: time.Unix(0, 0)}
	h.frame()
	h.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: f32.Pt(5, 5)})
	h.frame()
	h.frame()
	if !states[len(states)-1].Hovered {
		t.Error("content not told it's hovered")
	}
	h.r.Source().Execute(key.FocusCmd{Tag: p})
	h.frame()
	h.r.Queue(key.Event{Name: key.NameSpace, State: key.Press}, key.Event{Name: key.NameSpace, State: key.Release})
	h.frame()
	if len(events) != 1 || events[0].Source != ClickKeyboard {
		t.Fatalf("Space on focused pressable: %+v", events)
	}
	if !p.Focused() {
		t.Error("Focused() false after focusing")
	}
}

func TestPressable_Disabled(t *testing.T) {
	clicks := 0
	var last PressState
	p := NewPressable(func(s PressState) Element {
		last = s
		return sized(100, 40)
	}).OnClick(func(ClickEvent) { clicks++ }).Disabled(true)
	h := &harness{el: p, now: time.Unix(0, 0)}
	h.frame()
	h.click(10, 10, pointer.ButtonPrimary, 0)
	p.Click()
	if clicks != 0 || !last.Disabled {
		t.Errorf("disabled: %d clicks, state %+v", clicks, last)
	}
	p.Disabled(false).Click()
	if clicks != 1 {
		t.Error("programmatic click after enabling didn't fire")
	}
}

func TestBox_OnClick(t *testing.T) {
	var got ClickEvent
	b := Box(styles.Styles{Width: 100, Height: 40}).OnClick(func(e ClickEvent) { got = e })
	h := &harness{el: b, now: time.Unix(0, 0)}
	h.frame()
	h.click(10, 10, pointer.ButtonPrimary, 0)
	if got.Clicks != 1 {
		t.Errorf("Box.OnClick didn't fire: %+v", got)
	}
}

func TestPressable_TouchPressedAndReleasedOutside(t *testing.T) {
	clicks := 0
	var states []PressState
	p := NewPressable(func(s PressState) Element { states = append(states, s); return sized(100, 40) }).
		OnClick(func(ClickEvent) { clicks++ })
	h := &harness{el: p, now: time.Unix(0, 0)}
	h.frame()

	touch := func(kind pointer.Kind, x, y float32) {
		h.r.Queue(pointer.Event{Kind: kind, Source: pointer.Touch, Position: f32.Pt(x, y), Buttons: pointer.ButtonPrimary})
		h.frame()
	}

	touch(pointer.Press, 10, 10)
	h.frame()
	if !states[len(states)-1].Pressed {
		t.Error("PressState.Pressed not set during a touch")
	}
	touch(pointer.Release, 10, 10)
	if clicks != 1 {
		t.Fatalf("tap: %d clicks, want 1", clicks)
	}

	// The finger slides off the element before lifting: no click.
	touch(pointer.Press, 10, 10)
	touch(pointer.Move, 300, 300)
	touch(pointer.Release, 300, 300)
	if clicks != 1 {
		t.Errorf("released outside: %d clicks, want still 1", clicks)
	}
	if p.Pressed() {
		t.Error("still pressed after the release")
	}

	// Sliding off and back on still clicks.
	touch(pointer.Press, 10, 10)
	touch(pointer.Move, 300, 300)
	touch(pointer.Move, 20, 20)
	touch(pointer.Release, 20, 20)
	if clicks != 2 {
		t.Errorf("slid back before release: %d clicks, want 2", clicks)
	}
}

func TestPressable_LongPress(t *testing.T) {
	clicks, longs := 0, 0
	p := NewPressable(func(PressState) Element { return sized(100, 40) }).
		OnClick(func(ClickEvent) { clicks++ }).
		OnLongPress(func(ClickEvent) { longs++ })
	h := &harness{el: p, now: time.Unix(0, 0)}
	h.frame()
	pos := f32.Pt(10, 10)
	h.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Touch, Position: pos, Buttons: pointer.ButtonPrimary})
	h.frame()
	for i := 0; i < 40; i++ { // 640ms
		h.frame()
	}
	h.r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Touch, Position: pos})
	h.frame()
	if longs != 1 || clicks != 0 {
		t.Fatalf("held touch: %d long-presses, %d clicks; want 1, 0", longs, clicks)
	}
	h.click(10, 10, pointer.ButtonPrimary, 0)
	if longs != 1 || clicks != 1 {
		t.Errorf("then a click: %d long-presses, %d clicks; want 1, 1", longs, clicks)
	}
}

func TestClickWindowPosition(t *testing.T) {
	var got ClickEvent
	p := NewPressable(func(PressState) Element { return Box(styles.Styles{Width: 50, Height: 20}) }).
		OnClick(func(e ClickEvent) { got = e })
	var r input.Router
	frame := func() {
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Source: r.Source(), Constraints: layout.Exact(image.Pt(400, 300))}
		Stack(styles.Styles{},
			Box(styles.Styles{Height: 60}),
			Row(styles.Styles{}, Box(styles.Styles{Width: 100}), p),
		).Layout(gtx)
		r.Frame(&ops)
	}
	frame()
	frame()
	pos := f32.Pt(110, 65)
	r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: pos, Buttons: pointer.ButtonPrimary},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: pos})
	frame()
	if got.WindowPosition != image.Pt(110, 65) || got.Position != image.Pt(10, 5) {
		t.Errorf("click at %v: Position %v, WindowPosition %v", pos, got.Position, got.WindowPosition)
	}
}

// A press that moves is a drag - start, moves in dp, end - and its
// release isn't a click; a press that doesn't move still clicks.
func TestPressable_Drag(t *testing.T) {
	var clicks int
	var starts, moves, ends []DragEvent
	p := NewPressable(func(PressState) Element { return sized(100, 40) }).
		OnClick(func(ClickEvent) { clicks++ }).
		OnDragStart(func(e DragEvent) { starts = append(starts, e) }).
		OnDragMove(func(e DragEvent) { moves = append(moves, e) }).
		OnDragEnd(func(e DragEvent) { ends = append(ends, e) })
	// The pressable sits at (20, 30), at 2px per dp.
	h := &harness{el: Raw(func(gtx layout.Context) layout.Dimensions {
		gtx.Metric.PxPerDp = 2
		defer op.Offset(image.Pt(20, 30)).Push(gtx.Ops).Pop()
		pop := PushOffset(image.Pt(20, 30))
		defer pop()
		return p.Layout(gtx)
	}), now: time.Unix(0, 0)}
	h.frame()
	h.frame()

	at := func(x, y float32) f32.Point { return f32.Pt(x, y) }
	h.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: at(30, 40)})
	h.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: at(30, 40), Buttons: pointer.ButtonPrimary})
	h.frame()
	// Within the slop (6dp = 12px): not yet a drag.
	h.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: at(36, 40), Buttons: pointer.ButtonPrimary})
	h.frame()
	if len(starts) != 0 || p.Dragging() {
		t.Fatal("a move within the slop started a drag")
	}
	// Past it, and out of the pressable: the drag goes on.
	h.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: at(70, 40), Buttons: pointer.ButtonPrimary})
	h.frame()
	h.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: at(200, 140), Buttons: pointer.ButtonPrimary})
	h.frame()
	if len(starts) != 1 || !p.Dragging() || !p.State().Dragging {
		t.Fatalf("%d drag starts, dragging %v", len(starts), p.Dragging())
	}
	if s := starts[0]; s.Position != image.Pt(10, 10) || s.PositionDp != f32.Pt(5, 5) || s.WindowPosition != image.Pt(30, 40) || s.DeltaDp != (f32.Point{}) {
		t.Errorf("start = %+v, want the press position", s)
	}
	if len(moves) != 2 {
		t.Fatalf("%d moves, want 2", len(moves))
	}
	if m := moves[1]; m.WindowPosition != image.Pt(200, 140) || m.DeltaDp != f32.Pt(85, 50) || m.PositionDp != f32.Pt(90, 55) {
		t.Errorf("move = %+v", m)
	}
	h.r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: at(200, 140)})
	h.frame()
	if len(ends) != 1 || ends[0].Cancelled || ends[0].WindowPosition != image.Pt(200, 140) || p.Dragging() {
		t.Errorf("ends %+v, dragging %v", ends, p.Dragging())
	}
	if clicks != 0 {
		t.Errorf("the drag clicked %d times", clicks)
	}

	// Back over it, a press and release in place is a click.
	h.now = h.now.Add(time.Second)
	h.click(40, 50, pointer.ButtonPrimary, 0)
	if clicks != 1 || len(starts) != 1 {
		t.Errorf("a click: %d clicks, %d drag starts", clicks, len(starts))
	}

	// A cancelled drag ends cancelled.
	h.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: at(40, 50), Buttons: pointer.ButtonPrimary})
	h.frame()
	h.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: at(90, 50), Buttons: pointer.ButtonPrimary})
	h.frame()
	h.r.Queue(pointer.Event{Kind: pointer.Cancel, Source: pointer.Mouse})
	h.frame()
	if len(ends) != 2 || !ends[1].Cancelled || p.Dragging() || clicks != 1 {
		t.Errorf("after a cancel: ends %+v, dragging %v, clicks %d", ends, p.Dragging(), clicks)
	}
}

// A keyboard click's WindowPosition is the pressable's bottom-left
// corner, where a menu opened from it goes.
func TestPressable_KeyboardClickPosition(t *testing.T) {
	var got ClickEvent
	p := NewPressable(func(PressState) Element { return sized(100, 40) }).OnClick(func(e ClickEvent) { got = e })
	h := &harness{el: Raw(func(gtx layout.Context) layout.Dimensions {
		defer op.Offset(image.Pt(20, 30)).Push(gtx.Ops).Pop()
		pop := PushOffset(image.Pt(20, 30))
		defer pop()
		return p.Layout(gtx)
	}), now: time.Unix(0, 0)}
	h.frame()
	h.frame()
	p.Click()
	if got.WindowPosition != image.Pt(20, 70) {
		t.Errorf("programmatic click at %v, want (20, 70)", got.WindowPosition)
	}
}
