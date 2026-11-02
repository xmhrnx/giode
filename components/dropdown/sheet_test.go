package dropdown

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/mehran9675/giode/elements"
	"github.com/mehran9675/giode/fonts"
	"github.com/mehran9675/giode/styles"
)

type sheetHarness struct {
	r   input.Router
	s   *Select
	win image.Point
}

func (h *sheetHarness) frame() {
	elements.SetWindowSize(h.win)
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Source: h.r.Source(), Constraints: layout.Constraints{Max: h.win}}
	h.s.Layout(gtx)
	elements.DrainPortals(gtx)
	h.r.Frame(&ops)
}

func (h *sheetHarness) tap(x, y int) {
	pos := f32.Pt(float32(x), float32(y))
	h.r.Queue(
		pointer.Event{Kind: pointer.Press, Source: pointer.Touch, Position: pos, Buttons: pointer.ButtonPrimary},
		pointer.Event{Kind: pointer.Release, Source: pointer.Touch, Position: pos},
	)
	h.frame()
	h.frame()
}

func TestSelect_SheetPicksAndDismisses(t *testing.T) {
	defer elements.SetWindowSize(image.Point{})
	var chosen []int
	h := &sheetHarness{win: image.Pt(400, 800), s: New(styles.Styles{}, "Small", "Medium", "Large").Sheet(SheetAlways)}
	h.s.OnChange(func(i int) { chosen = append(chosen, i) })
	h.frame()

	h.tap(20, 10) // the field
	if !h.s.Opened() {
		t.Fatal("tapping the field didn't open the sheet")
	}
	// The last option sits just above the sheet's bottom padding.
	h.tap(200, 800-8-10)
	if h.s.Opened() || len(chosen) != 1 || chosen[0] != 2 {
		t.Fatalf("after tapping the last row: open=%v chosen=%v, want closed [2]", h.s.Opened(), chosen)
	}

	h.tap(20, 10)
	h.tap(200, 100) // the scrim, far above the sheet
	if h.s.Opened() || len(chosen) != 1 {
		t.Errorf("scrim tap: open=%v chosen=%v, want closed and no new choice", h.s.Opened(), chosen)
	}
}

func TestSelect_SheetCompactFollowsWindowWidth(t *testing.T) {
	defer elements.SetWindowSize(image.Point{})
	s := New(styles.Styles{}, "a").Sheet(SheetCompact)
	elements.SetWindowSize(image.Pt(1200, 800))
	if s.useSheet() {
		t.Error("sheet in a wide window")
	}
	elements.SetWindowSize(image.Pt(400, 800))
	if !s.useSheet() {
		t.Error("no sheet in a narrow window")
	}
	if New(styles.Styles{}, "a").useSheet() {
		t.Error("sheet without Sheet()")
	}
}

func TestSelect_PressOutsideCloses(t *testing.T) {
	fonts.UseGofont()
	defer elements.SetWindowSize(image.Point{})
	var chosen []int
	h := &sheetHarness{win: image.Pt(800, 600), s: New(styles.Styles{Width: 160}, "Small", "Medium", "Large").Sheet(SheetNever)}
	h.s.OnChange(func(i int) { chosen = append(chosen, i) })
	h.frame()
	click := func(x, y int, btn pointer.Buttons) {
		pos := f32.Pt(float32(x), float32(y))
		h.r.Queue(
			pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: pos, Buttons: btn},
			pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: pos},
		)
		h.frame()
		h.frame()
	}
	for _, btn := range []pointer.Buttons{pointer.ButtonPrimary, pointer.ButtonSecondary} {
		click(20, 10, pointer.ButtonPrimary)
		if !h.s.Opened() {
			t.Fatal("clicking the field didn't open the list")
		}
		click(600, 500, btn)
		if h.s.Opened() {
			t.Errorf("a press (buttons %v) outside the list left it open", btn)
		}
	}
	// Clicking the field again closes it too.
	click(20, 10, pointer.ButtonPrimary)
	click(20, 10, pointer.ButtonPrimary)
	if h.s.Opened() {
		t.Error("clicking the field of an open list didn't close it")
	}
	// Choosing still works.
	click(20, 10, pointer.ButtonPrimary)
	elements.BeginSemantics()
	h.frame()
	var medium image.Rectangle
	nodes := elements.EndSemantics()
	for _, n := range nodes {
		if n.Label == "Medium" && n.Kind != elements.SemanticSelect {
			medium = n.Bounds
		}
	}
	if medium.Empty() {
		t.Fatalf("no Medium option in the open list (open=%v): %+v", h.s.Opened(), nodes)
	}
	c := medium.Min.Add(medium.Max).Div(2)
	click(c.X, c.Y, pointer.ButtonPrimary)
	if h.s.Opened() || len(chosen) != 1 || chosen[0] != 1 {
		t.Errorf("picking an option: open=%v chosen=%v", h.s.Opened(), chosen)
	}
}

func TestSelect_SetOptions(t *testing.T) {
	changes := 0
	s := New(styles.Styles{}, "alpha", "beta", "gamma").OnChange(func(int) { changes++ })
	s.SetSelected(1)
	s.SetOptions("delta", "beta", "epsilon", "zeta")
	if s.Value() != "beta" || s.Selected() != 1 {
		t.Errorf("after SetOptions keeping beta: %q at %d", s.Value(), s.Selected())
	}
	s.SetOptions("one", "two")
	if s.Value() != "one" {
		t.Errorf("selection gone: %q, want the first option", s.Value())
	}
	s.SetOptions("a", "b", "c", "d", "e", "f")
	s.Choose(5)
	if s.Value() != "f" || changes != 1 {
		t.Errorf("Choose after growing: %q, %d changes", s.Value(), changes)
	}
	if len(s.Options()) != 6 {
		t.Errorf("options %v", s.Options())
	}
}
