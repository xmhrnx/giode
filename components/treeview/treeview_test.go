package treeview

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

	"github.com/xmhrnx/giode/elements"
)

type node struct {
	name     string
	kids     []Node
	lazy     bool // has children that load on expand
	loadable []Node
}

func (n *node) Key() any               { return n.name }
func (n *node) Children() []Node       { return n.kids }
func (n *node) HasChildren() bool      { return len(n.kids) > 0 || n.lazy }
func (n *node) Label() string          { return n.name }
func leaf(name string) *node           { return &node{name: name} }
func dir(name string, k ...Node) *node { return &node{name: name, kids: k} }

type harness struct {
	r     input.Router
	t     *Tree
	now   time.Time
	drawn []RowState
	names []string
}

func newHarness(roots []Node) *harness {
	h := &harness{now: time.Unix(0, 0)}
	h.t = New(func() []Node { return roots }, func(n Node, s RowState) elements.Element {
		h.drawn = append(h.drawn, s)
		h.names = append(h.names, n.(*node).name)
		return elements.Raw(func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: gtx.Constraints.Min} })
	})
	return h
}

func (h *harness) frame() {
	h.drawn, h.names = nil, nil
	h.now = h.now.Add(16 * time.Millisecond)
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Now: h.now, Source: h.r.Source(), Constraints: layout.Exact(image.Pt(300, 280))}
	elements.BeginFrame()
	h.t.Layout(gtx)
	elements.EndFrame()
	h.r.Frame(&ops)
}

func (h *harness) key(name key.Name) {
	h.r.Queue(key.Event{Name: name, State: key.Press})
	h.frame()
	h.frame()
}

func (h *harness) click(x, y float32, double bool) {
	pos := f32.Pt(x, y)
	n := 1
	if double {
		n = 2
	}
	for i := 0; i < n; i++ {
		h.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: pos})
		h.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: pos, Buttons: pointer.ButtonPrimary})
		h.frame()
		h.r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: pos})
		h.frame()
		h.frame() // the frame a handled click requests
	}
}

func sampleTree() []Node {
	return []Node{
		dir("src", dir("app", leaf("main.go"), leaf("ui.go")), leaf("go.mod")),
		dir("docs", leaf("index.md")),
		leaf("README"),
	}
}

func TestExpandCollapseAndDepth(t *testing.T) {
	h := newHarness(sampleTree())
	h.frame()
	if len(h.names) != 3 {
		t.Fatalf("collapsed tree rows = %v", h.names)
	}
	h.t.Expand("src", true)
	h.t.Expand("app", true)
	h.frame()
	want := []string{"src", "app", "main.go", "ui.go", "go.mod", "docs", "README"}
	if len(h.names) != len(want) {
		t.Fatalf("rows = %v, want %v", h.names, want)
	}
	for i, w := range want {
		if h.names[i] != w {
			t.Fatalf("rows = %v, want %v", h.names, want)
		}
	}
	if h.drawn[2].Depth != 2 || !h.drawn[1].Expanded || h.drawn[2].Expanded {
		t.Errorf("states: %+v", h.drawn[:3])
	}
	h.t.Toggle("src")
	h.frame()
	if len(h.names) != 3 {
		t.Errorf("after collapsing src: %v", h.names)
	}
}

func TestKeyboardNavigation(t *testing.T) {
	h := newHarness(sampleTree())
	var activated []string
	h.t.OnActivate(func(n Node) { activated = append(activated, n.(*node).name) })
	h.frame()
	h.r.Source().Execute(key.FocusCmd{Tag: h.t.list})
	h.frame()
	h.frame()
	sel := func() string { k, _ := h.t.Selected(); s, _ := k.(string); return s }
	if sel() != "src" {
		t.Fatalf("focusing selects the first row; got %q", sel())
	}
	h.key(key.NameRightArrow) // expands src
	if !h.t.Expanded("src") {
		t.Fatal("Right didn't expand")
	}
	h.key(key.NameRightArrow) // moves into src
	if sel() != "app" {
		t.Fatalf("second Right: selected %q, want app", sel())
	}
	h.key(key.NameDownArrow) // go.mod (app is collapsed)
	if sel() != "go.mod" {
		t.Fatalf("Down: %q, want go.mod", sel())
	}
	h.key(key.NameLeftArrow) // to parent src
	if sel() != "src" {
		t.Fatalf("Left on a leaf: %q, want its parent", sel())
	}
	h.key(key.NameLeftArrow) // collapses src
	if h.t.Expanded("src") {
		t.Fatal("Left on an open node didn't collapse it")
	}
	h.key(key.NameEnd)
	h.key(key.NameReturn)
	if sel() != "README" || len(activated) != 1 || activated[0] != "README" {
		t.Errorf("End + Enter: selected %q, activated %v", sel(), activated)
	}
}

func TestRevealOpensAncestors(t *testing.T) {
	h := newHarness(sampleTree())
	h.frame()
	h.t.Reveal("ui.go")
	h.frame()
	if !h.t.Expanded("src") || !h.t.Expanded("app") {
		t.Fatal("Reveal didn't open the ancestors")
	}
	if k, _ := h.t.Selected(); k != "ui.go" {
		t.Errorf("selected %v, want ui.go", k)
	}
}

func TestLazyLoading(t *testing.T) {
	folder := &node{name: "remote", lazy: true}
	h := newHarness([]Node{folder})
	expanded := 0
	h.t.OnExpand(func(n Node) { expanded++ })
	h.frame()
	h.t.Expand("remote", true)
	h.frame()
	if expanded != 1 || !h.drawn[0].Loading {
		t.Fatalf("expanding an unloaded folder: OnExpand %d, loading %v", expanded, h.drawn[0].Loading)
	}
	folder.kids = []Node{leaf("a"), leaf("b")}
	h.frame()
	if h.drawn[0].Loading || len(h.names) != 3 {
		t.Errorf("after loading: loading %v, rows %v", h.drawn[0].Loading, h.names)
	}
}

func TestClickArrowTogglesRowClickSelects(t *testing.T) {
	h := newHarness(sampleTree())
	var activated int
	h.t.OnActivate(func(Node) { activated++ })
	h.frame()
	// Row 0 ("src") spans y 0-28; its arrow sits right after the 4dp
	// padding, the content after the 20dp arrow box.
	h.click(12, 14, false)
	if !h.t.Expanded("src") {
		t.Fatal("clicking the arrow didn't expand")
	}
	if _, ok := h.t.Selected(); ok {
		t.Error("clicking the arrow selected the row")
	}
	h.now = h.now.Add(time.Second)
	h.click(150, 14+28*3, true) // "docs" (row 3 once src is open), double click
	if k, _ := h.t.Selected(); k != "docs" || activated != 1 {
		t.Errorf("double click on docs: selected %v, activated %d", k, activated)
	}
}
