package menu

import (
	"image"
	"strconv"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/mehran9675/giode/elements"
	"github.com/mehran9675/giode/fonts"
)

type menuHarness struct {
	t     *testing.T
	m     *Menu
	r     input.Router
	win   image.Point
	nodes map[string]image.Rectangle
}

func (h *menuHarness) frame() {
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Source: h.r.Source(), Constraints: layout.Exact(h.win)}
	elements.SetWindowSize(h.win)
	elements.BeginFrame()
	elements.BeginSemantics()
	h.m.Layout(gtx)
	elements.DrainPortals(gtx)
	h.nodes = map[string]image.Rectangle{}
	for _, n := range elements.EndSemantics() {
		h.nodes[n.Label] = n.Bounds
	}
	h.r.Frame(&ops)
}

func (h *menuHarness) key(name key.Name) {
	h.r.Queue(key.Event{Name: name, State: key.Press})
	h.frame()
	h.frame()
}

// visible reports whether the item labeled l is laid out inside the
// window.
func (h *menuHarness) visible(l string) bool {
	b, ok := h.nodes[l]
	return ok && b.Min.Y >= 0 && b.Max.Y <= h.win.Y
}

func longMenu(n int, chosen *string) *Menu {
	m := New()
	for i := range n {
		l := "Item " + strconv.Itoa(i)
		m.Item(l, func() { *chosen = l })
	}
	return m
}

func TestMenuScrollsWithKeyboard(t *testing.T) {
	fonts.UseGofont()
	defer elements.SetWindowSize(image.Point{})
	var chosen string
	h := &menuHarness{t: t, m: longMenu(40, &chosen), win: image.Pt(400, 300)}
	h.m.OpenAt(f32.Pt(10, 10))
	h.frame()
	h.frame()
	if h.visible("Item 30") {
		t.Fatal("a 40-item menu fits a 300px window: the test needs a taller menu")
	}
	for range 31 {
		h.key(key.NameDownArrow)
	}
	if h.m.active != 30 {
		t.Fatalf("after 31 Down presses the highlight is on %d, want 30", h.m.active)
	}
	if !h.visible("Item 30") {
		t.Errorf("the highlighted Item 30 isn't scrolled into view: %v", h.nodes["Item 30"])
	}
	h.key(key.NameReturn)
	if chosen != "Item 30" || h.m.Opened() {
		t.Errorf("Enter chose %q, open=%v; want Item 30 and closed", chosen, h.m.Opened())
	}
}

func TestMenuScrollsWithWheel(t *testing.T) {
	fonts.UseGofont()
	defer elements.SetWindowSize(image.Point{})
	var chosen string
	h := &menuHarness{t: t, m: longMenu(40, &chosen), win: image.Pt(400, 300)}
	h.m.OpenAt(f32.Pt(10, 10))
	h.frame()
	h.frame()
	first := h.nodes["Item 0"]
	for range 20 {
		h.r.Queue(pointer.Event{Kind: pointer.Scroll, Source: pointer.Mouse, Position: f32.Pt(40, 100), Scroll: f32.Pt(0, 200)})
		h.frame()
	}
	h.frame()
	if !h.visible("Item 39") {
		t.Errorf("scrolling down didn't bring the last item into view: %v", h.nodes["Item 39"])
	}
	if h.nodes["Item 0"].Min.Y >= first.Min.Y {
		t.Errorf("the first item didn't move up: %v, then %v", first, h.nodes["Item 0"])
	}
	if !h.m.Opened() {
		t.Error("scrolling closed the menu")
	}
}

func TestMenuKeyboardSubmenu(t *testing.T) {
	fonts.UseGofont()
	defer elements.SetWindowSize(image.Point{})
	var chosen string
	sub := New().Item("Deep", func() { chosen = "Deep" })
	m := New().Item("First", func() { chosen = "First" }).SubMenu("More", sub)
	h := &menuHarness{t: t, m: m, win: image.Pt(400, 300)}
	m.OpenAt(f32.Pt(10, 10))
	h.frame()
	h.key(key.NameDownArrow)
	h.key(key.NameDownArrow)
	h.key(key.NameRightArrow)
	if !sub.Opened() || sub.active != 0 {
		t.Fatalf("Right on More: submenu open=%v active=%d", sub.Opened(), sub.active)
	}
	h.key(key.NameLeftArrow)
	if sub.Opened() || !m.Opened() {
		t.Fatalf("Left: submenu open=%v, menu open=%v; want only the menu", sub.Opened(), m.Opened())
	}
	h.key(key.NameRightArrow)
	h.key(key.NameReturn)
	if chosen != "Deep" || m.Opened() {
		t.Errorf("Enter in the submenu chose %q, open=%v", chosen, m.Opened())
	}
}
