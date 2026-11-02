package input

import (
	"image"
	"io"
	"strings"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/io/transfer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/styles"
)

type menuHarness struct {
	r     input.Router
	in    *Input
	now   time.Time
	nodes []elements.SemanticNode
}

func (h *menuHarness) frame() {
	elements.BeginFrame()
	elements.BeginSemantics()
	elements.SetWindowSize(image.Pt(400, 300))
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Source: h.r.Source(), Now: h.now, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Constraints{Max: image.Pt(400, 300)}}
	// Room above the field for the menu.
	st := op.Offset(image.Pt(0, 100)).Push(gtx.Ops)
	pop := elements.PushOffset(image.Pt(0, 100))
	h.in.Layout(gtx)
	pop()
	st.Pop()
	elements.DrainPortals(gtx)
	h.nodes = elements.EndSemantics()
	h.r.Frame(&ops)
	elements.EndFrame()
}

func (h *menuHarness) touch(kind pointer.Kind, p image.Point) {
	h.r.Queue(pointer.Event{Kind: kind, Source: pointer.Touch, Position: f32.Pt(float32(p.X), float32(p.Y)), PointerID: 1})
	h.frame()
}

func (h *menuHarness) longPress(p image.Point) {
	h.touch(pointer.Press, p)
	h.now = h.now.Add(600 * time.Millisecond)
	h.frame()
	h.touch(pointer.Release, p)
}

// button returns the center of the menu button labelled label.
func (h *menuHarness) button(label string) (image.Point, bool) {
	for _, n := range h.nodes {
		if n.Label == label && n.Kind == elements.SemanticButton {
			return n.Bounds.Min.Add(n.Bounds.Max).Div(2), true
		}
	}
	return image.Point{}, false
}

// runeAt returns the window position of the middle of rune i.
func (h *menuHarness) runeAt(i int) image.Point {
	r := h.in.editor.Regions(i, i+1, nil)[0].Bounds
	return h.in.menu.origin.Add(r.Min.Add(r.Max).Div(2))
}

func TestInput_LongPressMenu(t *testing.T) {
	fonts.UseGofont()
	h := &menuHarness{in: New("", styles.Styles{Width: 400}), now: time.Unix(0, 0)}
	h.in.SetText("hello brave world")
	h.frame()
	h.frame()

	// Long-press on "brave" selects it and opens the menu.
	h.longPress(h.runeAt(8))
	if got := h.in.editor.SelectedText(); got != "brave" {
		t.Fatalf("long-press selected %q, want %q", got, "brave")
	}
	for _, l := range []string{"Cut", "Copy", "Paste", "Select all"} {
		if _, ok := h.button(l); !ok {
			t.Errorf("menu lacks %s", l)
		}
	}
	// The release didn't collapse the selection.
	h.frame()
	if got := h.in.editor.SelectedText(); got != "brave" {
		t.Errorf("after letting go: %q selected", got)
	}

	// Copy.
	p, _ := h.button("Copy")
	h.touch(pointer.Press, p)
	h.touch(pointer.Release, p)
	h.frame()
	if _, content, ok := h.r.WriteClipboard(); !ok || string(content) != "brave" {
		t.Errorf("clipboard %q (%v), want %q", content, ok, "brave")
	}
	if _, ok := h.button("Copy"); ok {
		t.Error("menu still open after Copy")
	}

	// The end handle extends the selection to "world".
	h.longPress(h.runeAt(8))
	end := h.in.menu.end.at
	target := h.runeAt(16)
	target.Y = end.Y
	grab := end.Add(image.Pt(0, 15))
	h.touch(pointer.Press, grab)
	h.touch(pointer.Move, grab.Add(image.Pt(10, 0)))
	h.touch(pointer.Move, grab.Add(target.Sub(end)).Add(image.Pt(6, 0)))
	h.touch(pointer.Release, grab.Add(target.Sub(end)).Add(image.Pt(6, 0)))
	if got := h.in.editor.SelectedText(); got != "brave world" {
		t.Errorf("after dragging the end handle: %q selected, want %q", got, "brave world")
	}

	// Paste replaces the selection.
	h.frame()
	p, ok := h.button("Paste")
	if !ok {
		t.Fatal("no Paste after dragging")
	}
	h.touch(pointer.Press, p)
	h.touch(pointer.Release, p)
	h.frame()
	if !h.r.ClipboardRequested() {
		t.Fatal("Paste didn't ask for the clipboard")
	}
	h.r.Queue(transfer.DataEvent{Type: "application/text", Open: func() io.ReadCloser { return io.NopCloser(strings.NewReader("there")) }})
	h.frame()
	h.frame()
	if got := h.in.Text(); got != "hello there" {
		t.Errorf("after Paste: %q, want %q", got, "hello there")
	}
}

func TestInput_LongPressEmptyFieldOffersPaste(t *testing.T) {
	fonts.UseGofont()
	h := &menuHarness{in: New("Type here", styles.Styles{Width: 400}), now: time.Unix(0, 0)}
	h.frame()
	h.frame()
	h.longPress(h.in.menu.origin.Add(image.Pt(50, 5)))
	if _, ok := h.button("Paste"); !ok {
		t.Error("no Paste on an empty field")
	}
	for _, l := range []string{"Cut", "Copy", "Select all"} {
		if _, ok := h.button(l); ok {
			t.Errorf("an empty field offers %s", l)
		}
	}
}

func TestInput_ScrollIsNotALongPress(t *testing.T) {
	fonts.UseGofont()
	h := &menuHarness{in: New("", styles.Styles{Width: 400}), now: time.Unix(0, 0)}
	h.in.SetText("hello")
	h.frame()
	h.frame()
	p := h.runeAt(1)
	h.touch(pointer.Press, p)
	h.touch(pointer.Move, p.Add(image.Pt(0, 30)))
	h.now = h.now.Add(time.Second)
	h.frame()
	if h.in.menu.open {
		t.Error("a moving finger opened the menu")
	}
}
