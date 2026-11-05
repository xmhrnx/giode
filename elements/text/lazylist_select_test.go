package text_test

import (
	"fmt"
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/components/lazylist"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/elements/text"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/styles"
)

type listHarness struct {
	r    input.Router
	root elements.Element
}

func (h *listHarness) frame() {
	elements.BeginFrame()
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Source: h.r.Source(), Constraints: layout.Exact(image.Pt(600, 300)), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
	h.root.Layout(gtx)
	h.r.Frame(&ops)
	elements.EndFrame()
}

func (h *listHarness) mouse(k pointer.Kind, x, y float32) {
	e := pointer.Event{Kind: k, Source: pointer.Mouse, Position: f32.Pt(x, y)}
	if k != pointer.Release {
		e.Buttons = pointer.ButtonPrimary
	}
	h.r.Queue(e)
	h.frame()
}

// Text in LazyList rows can be selected however the rows are made:
// Selectable text, a SelectionArea around the list or in a row, and
// rows that are clickable boxes - where a drag selects and a click
// still clicks.
func TestSelectTextInLazyListRows(t *testing.T) {
	fonts.UseGofont()
	clicks := 0
	rows := map[int]*elements.BoxElement{}
	clickable := func(i int) elements.Element {
		if rows[i] == nil {
			rows[i] = elements.Box(styles.Styles{Width: -1}, elements.Text(fmt.Sprintf("Row number %d text", i), styles.Styles{FontSize: 20})).
				OnClick(func(elements.ClickEvent) { clicks++ })
		}
		return rows[i]
	}
	type tc struct {
		item     func(i int) elements.Element
		wrap     bool
		selected func() string
	}
	cases := map[string]tc{
		"Selectable text": {
			item: func(i int) elements.Element {
				return elements.Text(fmt.Sprintf("Row number %d text", i), styles.Styles{FontSize: 20}).Selectable(fmt.Sprint("row", i))
			},
			selected: func() string { return text.SelectedTextOfSelectable("row0") },
		},
		"area around the list": {
			item: func(i int) elements.Element {
				return elements.Text(fmt.Sprintf("Row number %d text", i), styles.Styles{FontSize: 20})
			},
			wrap:     true,
			selected: func() string { return text.SelectedText("sel") },
		},
		"area in each row": {
			item: func(i int) elements.Element {
				return elements.SelectionArea(fmt.Sprint("area", i), elements.Text(fmt.Sprintf("Row number %d text", i), styles.Styles{FontSize: 20}))
			},
			selected: func() string { return text.SelectedText("area0") },
		},
		"clickable rows": {item: clickable, wrap: true, selected: func() string { return text.SelectedText("sel") }},
	}
	for name, c := range cases {
		for _, versioned := range []bool{false, true} {
			l := lazylist.New(func() int { return 50 }, func(i int) any { return i }, c.item)
			if versioned {
				l.Version(func(i int) any { return i })
			}
			h := &listHarness{root: l}
			if c.wrap {
				h.root = elements.SelectionArea("sel", l)
			}
			for i := 0; i < 3; i++ {
				h.frame()
			}
			clicks = 0
			h.mouse(pointer.Press, 2, 10)
			h.mouse(pointer.Move, 100, 12)
			h.mouse(pointer.Release, 100, 12)
			if got := c.selected(); got != "Row numb" {
				t.Errorf("%s (versioned %v): selected %q, want %q", name, versioned, got, "Row numb")
			}
			if clicks != 0 {
				t.Errorf("%s (versioned %v): the drag clicked the row", name, versioned)
			}
			if name == "clickable rows" {
				h.mouse(pointer.Press, 300, 10)
				h.mouse(pointer.Release, 300, 10)
				if clicks != 1 {
					t.Errorf("clickable rows (versioned %v): a click gave %d clicks, want 1", versioned, clicks)
				}
			}
		}
	}
}
