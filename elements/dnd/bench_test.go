package dnd_test

import (
	"fmt"
	"image"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/pointer"

	"github.com/xmhrnx/giode/components/lazylist"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/elements/dnd"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// kanban is 6 columns of cards (500 in all), each column a LazyList.
type kanban struct {
	cols  [6][]int
	cards map[int]*dnd.Draggable
	tgts  [6]*dnd.DropTarget
	lists [6]*lazylist.List
}

func newKanban(n int) *kanban {
	k := &kanban{cards: map[int]*dnd.Draggable{}}
	for i := range n {
		k.cols[i%6] = append(k.cols[i%6], i)
	}
	for c := range 6 {
		k.lists[c] = lazylist.New(func() int { return len(k.cols[c]) }, func(i int) any { return k.cols[c][i] }, func(i int) elements.Element {
			return dnd.ListItem(i, k.card(k.cols[c][i]))
		}, styles.Styles{Gap: 6})
		k.tgts[c] = dnd.NewDropTarget(func(s dnd.DropState) elements.Element {
			bg := "theme.surface"
			if s.Over {
				bg = "theme.border"
			}
			return elements.Stack(styles.Styles{Width: 220, Height: -1, Background: bg, Padding: properties.UniformInset(8), Gap: 8},
				elements.Text(fmt.Sprintf("Column %d", c+1), styles.Styles{FontWeight: properties.FontWeightBold}),
				elements.Box(styles.Styles{FlexGrow: 1}, k.lists[c]))
		}).List(dnd.AxisVertical).OnDrop(func(e dnd.DropEvent) {
			id := e.Payload.(int)
			for sc := range 6 {
				for i, v := range k.cols[sc] {
					if v == id {
						if sc == c {
							dnd.Move(k.cols[c], i, e.Index)
						} else {
							k.cols[sc] = dnd.Remove(k.cols[sc], i)
							k.cols[c] = dnd.Insert(k.cols[c], e.Index, id)
						}
						return
					}
				}
			}
		})
	}
	return k
}

func (k *kanban) card(id int) *dnd.Draggable {
	d := k.cards[id]
	if d == nil {
		d = dnd.NewDraggable(id, func(dnd.DragState) elements.Element {
			return elements.Stack(styles.Styles{Background: "theme.background", BorderRadius: 6, BorderWidth: 1, BorderColor: "theme.border", Padding: properties.UniformInset(10), Gap: 4},
				elements.Text(fmt.Sprintf("Card %d", id)),
				elements.Text("Follow up with the customer about the renewal", styles.Styles{FontSize: 12}))
		})
		k.cards[id] = d
	}
	return d
}

func (k *kanban) view() elements.Element {
	return elements.Row(styles.Styles{Gap: 12, Padding: properties.UniformInset(12)},
		k.tgts[0], k.tgts[1], k.tgts[2], k.tgts[3], k.tgts[4], k.tgts[5])
}

// BenchmarkKanbanDrag measures a frame of a drag across a 500-card
// board: the pointer moves every frame, over a column's cards.
func BenchmarkKanbanDrag(b *testing.B) {
	k := newKanban(500)
	h := &harness{view: k.view, now: time.Unix(0, 0), size: image.Pt(1440, 900)}
	h.frame()
	h.frame()
	start := f32.Pt(100, 80)
	h.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: start})
	h.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: start, Buttons: pointer.ButtonPrimary})
	h.frame()
	h.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: start.Add(f32.Pt(20, 20)), Buttons: pointer.ButtonPrimary})
	h.frame()
	if !dnd.Dragging() {
		b.Fatal("drag didn't start")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Back and forth over the third column's cards, away from its
		// edges so it doesn't scroll.
		y := float32(200 + (i*7)%400)
		h.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: f32.Pt(560, y), Buttons: pointer.ButtonPrimary})
		h.frame()
	}
	b.StopTimer()
	h.r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: f32.Pt(560, 300)})
	h.frame()
}

// BenchmarkKanbanIdle is the same board's frame without a drag, for
// comparison.
func BenchmarkKanbanIdle(b *testing.B) {
	k := newKanban(500)
	h := &harness{view: k.view, now: time.Unix(0, 0), size: image.Pt(1440, 900)}
	h.frame()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: f32.Pt(560, float32(200+(i*7)%400))})
		h.frame()
	}
}

// TestKanbanDrop moves a card between LazyList columns of the board.
func TestKanbanDrop(t *testing.T) {
	k := newKanban(60)
	h := &harness{view: k.view, now: time.Unix(0, 0), size: image.Pt(1440, 900)}
	h.frame()
	h.frame()
	h.drag(f32.Pt(100, 80), f32.Pt(120, 100), f32.Pt(560, 70))
	if len(k.cols[0]) != 9 || len(k.cols[2]) != 11 || k.cols[2][0] != 0 {
		t.Errorf("columns 0 %v, 2 %v", k.cols[0], k.cols[2])
	}
}
