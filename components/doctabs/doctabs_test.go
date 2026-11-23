package doctabs

import (
	"fmt"
	"image"
	"slices"
	"strings"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/components/menu"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/fonts"
)

type harness struct {
	r     input.Router
	now   time.Time
	d     *DocTabs
	width int
	nodes []elements.SemanticNode
}

func newHarness(d *DocTabs, width int) *harness {
	fonts.UseGofont()
	h := &harness{d: d, now: time.Unix(0, 0), width: width}
	h.frame()
	h.frame()
	return h
}

func (h *harness) frame() {
	elements.BeginFrame()
	defer elements.EndFrame()
	elements.BeginSemantics()
	h.now = h.now.Add(16 * time.Millisecond)
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Now: h.now, Source: h.r.Source(), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Constraints{Max: image.Pt(h.width, 400)}}
	elements.SetWindowSize(image.Pt(h.width, 400))
	h.d.Layout(gtx)
	elements.DrainContextMenus(gtx)
	elements.DrainPortals(gtx)
	h.nodes = elements.EndSemantics()
	h.r.Frame(&ops)
}

func (h *harness) mouse(kind pointer.Kind, b pointer.Buttons, x float32) {
	h.r.Queue(pointer.Event{Kind: kind, Source: pointer.Mouse, Buttons: b, Position: f32.Pt(x, 17)})
}

func (h *harness) click(b pointer.Buttons, x float32) {
	h.mouse(pointer.Move, 0, x)
	h.mouse(pointer.Press, b, x)
	h.mouse(pointer.Release, 0, x)
	h.frame()
	h.frame()
}

// slot returns the tab's place in the last frame.
func (h *harness) slot(id string) slot {
	for _, s := range h.d.slots {
		if s.id == id {
			return s
		}
	}
	return slot{}
}

func (h *harness) node(label string) (elements.SemanticNode, bool) {
	for _, n := range h.nodes {
		if n.Label == label && n.Kind != elements.SemanticText {
			return n, true
		}
	}
	return elements.SemanticNode{}, false
}

func ids(d *DocTabs) string {
	var s []string
	for _, t := range d.Tabs() {
		s = append(s, t.ID)
	}
	return strings.Join(s, ",")
}

func threeTabs() *DocTabs {
	return New().
		Add(Tab{ID: "a", Title: "alpha.go"}).
		Add(Tab{ID: "b", Title: "beta.go"}).
		Add(Tab{ID: "c", Title: "gamma.go"})
}

func TestSelectAndClose(t *testing.T) {
	var selected, closed []string
	d := threeTabs().
		OnSelect(func(id string) { selected = append(selected, id) }).
		OnClose(func(id string) { closed = append(closed, id) })
	d.Select("a")
	h := newHarness(d, 600)

	b := h.slot("b")
	h.click(pointer.ButtonPrimary, float32(b.r.Min.X+10))
	if d.Selected() != "b" || !slices.Equal(selected, []string{"b"}) {
		t.Fatalf("selected %q, reported %v", d.Selected(), selected)
	}
	// The close button of the hovered tab asks; the tab stays until the
	// app removes it.
	c := h.slot("c")
	h.click(pointer.ButtonPrimary, float32(c.close.Min.X+5))
	if !slices.Equal(closed, []string{"c"}) || d.Len() != 3 || d.Selected() != "b" {
		t.Fatalf("close button: closed %v, %d tabs, selected %q", closed, d.Len(), d.Selected())
	}
	// A middle click closes too.
	h.click(pointer.ButtonTertiary, float32(h.slot("a").r.Min.X+10))
	if !slices.Equal(closed, []string{"c", "a"}) {
		t.Fatalf("middle click: closed %v", closed)
	}
	// Removing the selected tab selects the one after it.
	d.Remove("b")
	if d.Selected() != "c" || selected[len(selected)-1] != "c" || ids(d) != "a,c" {
		t.Errorf("after removing b: selected %q, tabs %s", d.Selected(), ids(d))
	}
	d.Remove("c")
	if d.Selected() != "a" {
		t.Errorf("after removing the last: selected %q", d.Selected())
	}
}

// Without OnClose, closing removes the tab.
func TestCloseWithoutHandler(t *testing.T) {
	d := threeTabs()
	d.Select("b")
	h := newHarness(d, 600)
	b := h.slot("b")
	h.click(pointer.ButtonPrimary, float32(b.close.Min.X+5))
	if ids(d) != "a,c" || d.Selected() != "c" {
		t.Errorf("tabs %s, selected %q", ids(d), d.Selected())
	}
}

func TestRenameModifiedAndSemantics(t *testing.T) {
	d := threeTabs()
	d.Select("a")
	d.SetModified("b", true)
	d.SetTitle("c", "renamed.go")
	d.Rename("a", "a2")
	h := newHarness(d, 600)
	if n, ok := h.node("beta.go"); !ok || !strings.Contains(n.Value, "modified") {
		t.Errorf("modified tab node: %+v %v", n, ok)
	}
	if _, ok := h.node("renamed.go"); !ok {
		t.Error("renamed tab missing")
	}
	if d.Selected() != "a2" {
		t.Errorf("selection after Rename: %q", d.Selected())
	}
	n, ok := h.node("Close renamed.go")
	if !ok {
		t.Fatal("no close button node")
	}
	n.Target.(interface{ Click() }).Click()
	if ids(d) != "a2,b" {
		t.Errorf("after automation close: %s", ids(d))
	}
}

func TestNewButton(t *testing.T) {
	n := 0
	d := threeTabs().OnNew(func() { n++ })
	h := newHarness(d, 600)
	r := h.d.newRect
	if r.Empty() || r.Min.X != h.d.total {
		t.Fatalf("the + button at %v, tabs end at %d", r, h.d.total)
	}
	h.click(pointer.ButtonPrimary, float32(r.Min.X+r.Dx()/2))
	if n != 1 {
		t.Errorf("+ clicked %d times", n)
	}
}

// Tabs that don't fit shrink to their minimum, then scroll; the
// selected one is scrolled into view.
func TestOverflow(t *testing.T) {
	d := New().OnNew(func() {})
	for i := range 20 {
		d.Add(Tab{ID: fmt.Sprint(i), Title: fmt.Sprintf("a rather long file name %d.go", i)})
	}
	d.Select("0")
	h := newHarness(d, 600)
	for _, s := range h.d.slots {
		if s.r.Dx() != 96 {
			t.Fatalf("tab %s is %d wide, want the minimum 96", s.id, s.r.Dx())
		}
		if !s.truncated {
			t.Fatalf("tab %s isn't cut off", s.id)
		}
	}
	if h.d.newRect.Max.X != 600 {
		t.Errorf("the + button isn't pinned to the end: %v", h.d.newRect)
	}
	d.Select("19")
	h.frame()
	if s := h.slot("19"); s.r.Max.X != h.d.tabsW {
		t.Errorf("the selected last tab ends at %d, want %d", s.r.Max.X, h.d.tabsW)
	}
	// The wheel scrolls back.
	h.r.Queue(pointer.Event{Kind: pointer.Scroll, Source: pointer.Mouse, Position: f32.Pt(100, 17), Scroll: f32.Pt(0, -10000)})
	h.frame()
	if h.d.scrollX != 0 {
		t.Errorf("scrolled to %d", h.d.scrollX)
	}
	// Fewer tabs shrink but don't scroll.
	d2 := New()
	for i := range 5 {
		d2.Add(Tab{ID: fmt.Sprint(i), Title: strings.Repeat("long ", 20)})
	}
	h2 := newHarness(d2, 600)
	if h2.d.total != 600 {
		t.Errorf("five shrunk tabs take %d, want 600", h2.d.total)
	}
}

// The full title shows in a tooltip when cut off, after a rest.
func TestTooltip(t *testing.T) {
	d := New().Add(Tab{ID: "x", Title: strings.Repeat("very long name ", 10)}).Add(Tab{ID: "y", Title: "short", Tooltip: "/home/me/short"})
	h := newHarness(d, 600)
	h.mouse(pointer.Move, 0, float32(h.slot("y").r.Min.X+5))
	h.frame()
	shown := func() bool {
		for _, n := range h.nodes {
			if n.Label == "/home/me/short" {
				return true
			}
		}
		return false
	}
	if shown() {
		t.Fatal("tooltip at once")
	}
	h.now = h.now.Add(time.Second)
	h.frame()
	if !shown() {
		t.Error("no tooltip after resting")
	}
}

func TestDragReorder(t *testing.T) {
	var order []string
	d := threeTabs().OnReorder(func(ids []string) { order = ids })
	h := newHarness(d, 600)
	a, b := h.slot("a"), h.slot("b")
	x := float32(a.r.Min.X + 10)
	h.mouse(pointer.Move, 0, x)
	h.mouse(pointer.Press, pointer.ButtonPrimary, x)
	h.frame()
	// a's middle past b's.
	to := x + float32(b.r.Min.X+b.r.Dx()/2-(a.r.Min.X+a.r.Dx()/2)) + 5
	for _, p := range []float32{x + 10, (x + to) / 2, to} {
		h.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: f32.Pt(p, 17)})
		h.frame()
	}
	if ids(d) != "b,a,c" {
		t.Fatalf("while dragging: %s", ids(d))
	}
	h.mouse(pointer.Release, 0, to)
	h.frame()
	if !slices.Equal(order, []string{"b", "a", "c"}) {
		t.Errorf("OnReorder got %v", order)
	}
	if d.Selected() != "a" {
		t.Errorf("the dragged tab isn't selected: %q", d.Selected())
	}
}

func TestTabMenu(t *testing.T) {
	var closed [][]string
	extra := ""
	d := threeTabs().
		OnCloseTabs(func(ids []string) { closed = append(closed, ids) }).
		OnClose(func(id string) { closed = append(closed, []string{id}) }).
		TabMenu(func(id string, m *menu.Menu) { m.Separator().Item("Copy Path", func() { extra = id }) })
	h := newHarness(d, 600)
	defer d.menu.Close()
	h.click(pointer.ButtonSecondary, float32(h.slot("b").r.Min.X+10))
	if !d.menu.Opened() {
		t.Fatal("no menu")
	}
	click := func(label string) {
		t.Helper()
		n, ok := h.node(label)
		if !ok {
			t.Fatalf("no %q item", label)
		}
		n.Target.(interface{ Click() }).Click()
	}
	click("Close Others")
	click("Close to the Right")
	click("Copy Path")
	want := [][]string{{"a", "c"}, {"c"}}
	if fmt.Sprint(closed) != fmt.Sprint(want) || extra != "b" {
		t.Errorf("closed %v (want %v), extra item for %q", closed, want, extra)
	}
}

func TestFit(t *testing.T) {
	for _, c := range []struct {
		nat         []int
		avail, minW int
		want        []int
	}{
		{[]int{100, 100}, 300, 50, []int{100, 100}},
		{[]int{60, 200, 200}, 300, 50, []int{60, 120, 120}},
		{[]int{200, 200, 200}, 90, 50, []int{50, 50, 50}},
	} {
		if got := fit(c.nat, c.avail, c.minW); !slices.Equal(got, c.want) {
			t.Errorf("fit(%v, %d, %d) = %v, want %v", c.nat, c.avail, c.minW, got, c.want)
		}
	}
}

func TestTabIconWidensTab(t *testing.T) {
	plain, iconed := New(), New()
	plain.Add(Tab{ID: "a", Title: "configuration_loader.go"})
	iconed.Add(Tab{ID: "a", Title: "configuration_loader.go", Icon: elements.Raw(func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: gtx.Constraints.Max}
	})})
	gtx := layout.Context{Ops: new(op.Ops), Constraints: layout.Constraints{Max: image.Pt(1<<20, 40)}, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
	w1 := plain.Layout(gtx).Size.X
	gtx.Ops.Reset()
	w2 := iconed.Layout(gtx).Size.X
	if w2 != w1+iconSize+iconGap {
		t.Errorf("width with icon %d, want %d", w2, w1+iconSize+iconGap)
	}
	iconed.SetIcon("a", nil)
	if tab, _ := iconed.Tab("a"); tab.Icon != nil {
		t.Error("SetIcon(nil) kept the icon")
	}
}

// Pinned tabs come first, keep a pin for a close button (clicking it
// unpins), and outlast middle clicks and Close Others.
func TestPinnedTabs(t *testing.T) {
	var closed [][]string
	var pins []string
	d := threeTabs().
		OnCloseTabs(func(ids []string) { closed = append(closed, ids) }).
		OnClose(func(id string) { closed = append(closed, []string{id}) }).
		OnPin(func(id string, pinned bool) { pins = append(pins, fmt.Sprintf("%s %v", id, pinned)) })
	d.SetPinned("c", true)
	d.Add(Tab{ID: "p", Title: "pinned.go", Pinned: true})
	if ids(d) != "c,p,a,b" {
		t.Fatalf("order %s, want the pinned first", ids(d))
	}
	h := newHarness(d, 800)
	defer d.menu.Close()
	h.click(pointer.ButtonTertiary, float32(h.slot("c").r.Min.X+10))
	if len(closed) != 0 {
		t.Fatalf("a middle click closed a pinned tab: %v", closed)
	}
	if n, ok := h.node("Unpin gamma.go"); !ok || n.Kind != elements.SemanticButton {
		t.Error("no Unpin button on the pinned tab")
	}
	h.click(pointer.ButtonSecondary, float32(h.slot("a").r.Min.X+10))
	n, ok := h.node("Close Others")
	if !ok {
		t.Fatal("no Close Others")
	}
	n.Target.(interface{ Click() }).Click()
	if fmt.Sprint(closed) != "[[b]]" {
		t.Errorf("Close Others closed %v, want [[b]]", closed)
	}
	d.menu.Close()
	h.frame()
	// The pin unpins.
	h.click(pointer.ButtonPrimary, float32(h.slot("c").close.Min.X+5))
	if d.pinned("c") || ids(d) != "p,c,a,b" || fmt.Sprint(pins) != "[c false]" {
		t.Errorf("after clicking the pin: pinned %v, order %s, OnPin %v", d.pinned("c"), ids(d), pins)
	}
}
