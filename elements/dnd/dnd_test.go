package dnd_test

import (
	"fmt"
	"image"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/components/lazylist"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/elements/dnd"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
	"github.com/xmhrnx/giode/ui"
)

type harness struct {
	r    input.Router
	view func() elements.Element
	now  time.Time
	size image.Point
}

func (h *harness) frame() {
	var ops op.Ops
	h.now = h.now.Add(16 * time.Millisecond)
	size := h.size
	if size == (image.Point{}) {
		size = image.Pt(400, 400)
	}
	gtx := layout.Context{Ops: &ops, Now: h.now, Constraints: layout.Exact(size), Source: h.r.Source()}
	ui.Frame(gtx, elements.Raw(func(gtx layout.Context) layout.Dimensions { return h.view().Layout(gtx) }))
	h.r.Frame(&ops)
}

func (h *harness) mouse(kind pointer.Kind, x, y float32) {
	e := pointer.Event{Kind: kind, Source: pointer.Mouse, Position: f32.Pt(x, y)}
	if kind == pointer.Press || kind == pointer.Move {
		e.Buttons = pointer.ButtonPrimary
	}
	if kind == pointer.Move {
		e.Buttons = 0
	}
	h.r.Queue(e)
}

// drag presses at from, moves to each point (a frame each) and
// releases at the last.
func (h *harness) drag(from f32.Point, to ...f32.Point) {
	h.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: from})
	h.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: from, Buttons: pointer.ButtonPrimary})
	h.frame()
	for _, p := range to {
		h.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: p, Buttons: pointer.ButtonPrimary})
		h.frame()
		h.frame()
	}
	last := to[len(to)-1]
	h.r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: last})
	h.frame()
	h.frame()
}

func sized(w, h int) elements.Element {
	return elements.Box(styles.Styles{Width: properties.Width(w), Height: properties.Height(h)})
}

// board is two columns of 40px cards, 100px wide, side by side.
type board struct {
	cols  [2][]string
	cards map[string]*dnd.Draggable
	tgts  [2]*dnd.DropTarget
	drops []dnd.DropEvent
	ends  []dnd.DragResult
}

func newBoard() *board {
	b := &board{cards: map[string]*dnd.Draggable{}}
	b.cols[0] = []string{"a", "b", "c"}
	b.cols[1] = []string{"x", "y"}
	for c := range 2 {
		b.tgts[c] = dnd.NewDropTarget(func(dnd.DropState) elements.Element {
			var items []elements.Element
			for i, id := range b.cols[c] {
				items = append(items, dnd.ListItem(i, b.card(id)))
			}
			return elements.Stack(styles.Styles{Width: 100, Height: 300}, items...)
		}).List(dnd.AxisVertical).Accept(dnd.AcceptType[string]()).OnDrop(func(e dnd.DropEvent) {
			b.drops = append(b.drops, e)
			id := e.Payload.(string)
			for sc := range 2 {
				for i, v := range b.cols[sc] {
					if v != id {
						continue
					}
					if sc == c {
						dnd.Move(b.cols[c], i, e.Index)
						return
					}
					b.cols[sc] = dnd.Remove(b.cols[sc], i)
					b.cols[c] = dnd.Insert(b.cols[c], e.Index, id)
					return
				}
			}
		})
	}
	return b
}

func (b *board) card(id string) *dnd.Draggable {
	d := b.cards[id]
	if d == nil {
		d = dnd.NewDraggable(id, func(dnd.DragState) elements.Element { return sized(100, 40) }).
			OnDragEnd(func(r dnd.DragResult) { b.ends = append(b.ends, r) })
		b.cards[id] = d
	}
	return d
}

func (b *board) view() elements.Element {
	return elements.Row(styles.Styles{}, b.tgts[0], b.tgts[1])
}

func TestDragBetweenLists(t *testing.T) {
	b := newBoard()
	h := &harness{view: b.view, now: time.Unix(0, 0)}
	h.frame()
	// "a" (column 0, y 0-40) to between "x" and "y" (column 1, y 40).
	h.drag(f32.Pt(50, 20), f32.Pt(60, 30), f32.Pt(150, 45))
	if len(b.drops) != 1 || b.drops[0].Index != 1 || b.drops[0].Payload != "a" {
		t.Fatalf("drops %+v, want a at slot 1", b.drops)
	}
	if fmt.Sprint(b.cols) != "[[b c] [x a y]]" {
		t.Errorf("columns %v", b.cols)
	}
	if len(b.ends) != 1 || !b.ends[0].Dropped || b.ends[0].Target != b.tgts[1] {
		t.Errorf("end %+v", b.ends)
	}
	if dnd.Dragging() {
		t.Error("still dragging")
	}
}

func TestReorderWithinList(t *testing.T) {
	b := newBoard()
	h := &harness{view: b.view, now: time.Unix(0, 0)}
	h.frame()
	// "a" below "c": past c's middle (100).
	h.drag(f32.Pt(50, 20), f32.Pt(50, 40), f32.Pt(50, 110))
	if fmt.Sprint(b.cols[0]) != "[b c a]" {
		t.Errorf("column %v drops %+v ends %+v", b.cols[0], b.drops, b.ends)
	}
	// "a" back above "b".
	h.drag(f32.Pt(50, 100), f32.Pt(50, 80), f32.Pt(50, 10))
	if fmt.Sprint(b.cols[0]) != "[a b c]" {
		t.Errorf("column %v", b.cols[0])
	}
}

func TestDropOutsideCancels(t *testing.T) {
	b := newBoard()
	h := &harness{view: b.view, now: time.Unix(0, 0)}
	h.frame()
	h.drag(f32.Pt(50, 20), f32.Pt(60, 30), f32.Pt(300, 350))
	if len(b.drops) != 0 || len(b.ends) != 1 || !b.ends[0].Cancelled {
		t.Errorf("drops %+v ends %+v", b.drops, b.ends)
	}
}

func TestEscapeCancels(t *testing.T) {
	b := newBoard()
	h := &harness{view: b.view, now: time.Unix(0, 0)}
	h.frame()
	h.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: f32.Pt(50, 20), Buttons: pointer.ButtonPrimary})
	h.frame()
	h.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: f32.Pt(150, 30), Buttons: pointer.ButtonPrimary})
	h.frame()
	h.frame()
	if !dnd.Dragging() {
		t.Fatal("drag didn't start")
	}
	h.r.Queue(key.Event{Name: key.NameEscape, State: key.Press})
	h.frame()
	if dnd.Dragging() {
		t.Fatal("Escape didn't cancel")
	}
	h.r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: f32.Pt(150, 30)})
	h.frame()
	if len(b.drops) != 0 {
		t.Errorf("dropped after cancelling: %+v", b.drops)
	}
}

func TestTapKeepsContentClick(t *testing.T) {
	clicks := 0
	p := elements.NewPressable(func(elements.PressState) elements.Element { return sized(100, 40) }).
		OnClick(func(elements.ClickEvent) { clicks++ })
	d := dnd.NewDraggable("p", func(dnd.DragState) elements.Element { return p })
	h := &harness{view: func() elements.Element { return d }, now: time.Unix(0, 0)}
	h.frame()
	h.drag(f32.Pt(10, 10), f32.Pt(11, 11))
	if clicks != 1 {
		t.Errorf("clicks = %d, want a tap to click", clicks)
	}
	h.drag(f32.Pt(10, 10), f32.Pt(60, 30), f32.Pt(80, 30))
	if clicks != 1 {
		t.Errorf("clicks = %d, a drag clicked", clicks)
	}
}

func TestTouchStartsWithLongPress(t *testing.T) {
	b := newBoard()
	h := &harness{view: b.view, now: time.Unix(0, 0)}
	h.frame()
	touch := func(kind pointer.Kind, x, y float32) {
		h.r.Queue(pointer.Event{Kind: kind, Source: pointer.Touch, Position: f32.Pt(x, y), PointerID: 1})
	}
	// Moving at once is a scroll, not a drag.
	touch(pointer.Press, 50, 20)
	h.frame()
	touch(pointer.Move, 50, 60)
	h.frame()
	if dnd.Dragging() {
		t.Fatal("a touch moving at once started a drag")
	}
	touch(pointer.Release, 50, 60)
	h.frame()

	touch(pointer.Press, 50, 20)
	h.frame()
	for range 40 {
		h.frame()
	}
	if !dnd.Dragging() {
		t.Fatal("a long press didn't start a drag")
	}
	touch(pointer.Move, 150, 10)
	h.frame()
	h.frame()
	touch(pointer.Release, 150, 10)
	h.frame()
	if len(b.drops) != 1 || b.drops[0].Index != 0 || fmt.Sprint(b.cols) != "[[b c] [a x y]]" {
		t.Errorf("drops %+v, columns %v", b.drops, b.cols)
	}
}

func TestKeyboardDrag(t *testing.T) {
	b := newBoard()
	h := &harness{view: b.view, now: time.Unix(0, 0)}
	h.frame()
	press := func(n key.Name) {
		h.r.Queue(key.Event{Name: n, State: key.Press})
		h.frame()
		h.frame()
	}
	// Focus "a"; Space picks it up.
	h.r.Source().Execute(key.FocusCmd{Tag: b.cards["a"]})
	h.frame()
	press(key.NameSpace)
	if !dnd.Dragging() {
		t.Fatal("Space didn't pick it up")
	}
	// Down once: after "b" (slot 2, stepping over slot 1, its own place).
	press(key.NameDownArrow)
	press(key.NameSpace)
	if fmt.Sprint(b.cols[0]) != "[b a c]" {
		t.Fatalf("column %v, drops %+v", b.cols[0], b.drops)
	}
	if !b.drops[0].Keyboard {
		t.Error("not a keyboard drop")
	}
	// "b" to the top of the other column with Right, then drop.
	h.r.Source().Execute(key.FocusCmd{Tag: b.cards["b"]})
	h.frame()
	press(key.NameSpace)
	press(key.NameRightArrow)
	press(key.NameReturn)
	if len(b.drops) != 2 || fmt.Sprint(b.cols) != "[[a c] [b x y]]" {
		t.Errorf("drops %+v, columns %v", b.drops, b.cols)
	}
}

// A drag held at the bottom edge of a LazyList scrolls it, and drops
// on rows that weren't laid out when it started.
func TestAutoScrollVirtualized(t *testing.T) {
	items := make([]int, 200)
	for i := range items {
		items[i] = i
	}
	drags := map[int]*dnd.Draggable{}
	var drop *dnd.DropEvent
	list := lazylist.New(func() int { return len(items) }, func(i int) any { return items[i] }, func(i int) elements.Element {
		id := items[i]
		d := drags[id]
		if d == nil {
			d = dnd.NewDraggable(id, func(dnd.DragState) elements.Element { return sized(200, 40) })
			drags[id] = d
		}
		return dnd.ListItem(i, d)
	})
	tgt := dnd.NewDropTarget(func(dnd.DropState) elements.Element { return list }).List(dnd.AxisVertical).
		OnDrop(func(e dnd.DropEvent) { drop = &e })
	h := &harness{view: func() elements.Element { return tgt }, now: time.Unix(0, 0)}
	h.frame()
	h.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: f32.Pt(50, 20)})
	h.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: f32.Pt(50, 20), Buttons: pointer.ButtonPrimary})
	h.frame()
	h.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: f32.Pt(50, 395), Buttons: pointer.ButtonPrimary})
	for range 120 {
		h.frame()
	}
	h.r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: f32.Pt(50, 395)})
	h.frame()
	if drop == nil || drop.Payload != 0 {
		t.Fatalf("drop %+v", drop)
	}
	if drop.Index < 40 {
		t.Errorf("dropped at slot %d: the list didn't scroll", drop.Index)
	}
}

// A target inside a Memo keeps registering while a drag runs.
func TestTargetInsideMemo(t *testing.T) {
	b := newBoard()
	view := func() elements.Element {
		return elements.Row(styles.Styles{},
			elements.Memo("c0", func() elements.Element { return b.tgts[0] }),
			elements.Memo("c1", func() elements.Element { return b.tgts[1] }))
	}
	h := &harness{view: view, now: time.Unix(0, 0)}
	h.frame()
	h.frame()
	h.drag(f32.Pt(50, 20), f32.Pt(60, 30), f32.Pt(150, 45))
	if len(b.drops) != 1 || b.drops[0].Index != 1 {
		t.Fatalf("drops %+v", b.drops)
	}
}

func TestMove(t *testing.T) {
	s := []int{0, 1, 2, 3}
	if got := dnd.Move(s, 0, 4); got != 3 || fmt.Sprint(s) != "[1 2 3 0]" {
		t.Errorf("got %d, %v", got, s)
	}
	if got := dnd.Move(s, 3, 0); got != 0 || fmt.Sprint(s) != "[0 1 2 3]" {
		t.Errorf("got %d, %v", got, s)
	}
	if got := dnd.Move(s, 1, 2); got != 1 || fmt.Sprint(s) != "[0 1 2 3]" {
		t.Errorf("no-op move: got %d, %v", got, s)
	}
}
