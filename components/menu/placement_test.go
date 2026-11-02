package menu

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/styles"
)

// A menu laid out deep in a page, under a header, opens where it's told,
// takes no space from its siblings, and closes on a press anywhere in
// the window - not just inside the area it was laid out in.
func TestMenuInLayout(t *testing.T) {
	fonts.UseGofont()
	defer elements.SetWindowSize(image.Point{})
	m := New().Item("Rename", func() {}).Item("Delete", func() {})
	win := image.Pt(600, 400)
	var r input.Router
	var list layout.Dimensions
	var nodes []elements.SemanticNode
	frame := func() {
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Source: r.Source(), Constraints: layout.Exact(win)}
		elements.SetWindowSize(win)
		elements.BeginFrame()
		elements.BeginSemantics()
		elements.Stack(styles.Styles{Width: -1, Height: -1},
			elements.Box(styles.Styles{Width: -1, Height: 40}), // a title bar
			elements.Row(styles.Styles{Width: -1},
				elements.Box(styles.Styles{Width: 200, Height: 300},
					elements.Stack(styles.Styles{Gap: 8},
						elements.Raw(func(gtx layout.Context) layout.Dimensions {
							list = elements.Box(styles.Styles{Width: 180, Height: 120}).Layout(gtx)
							return list
						}),
						m,
					)),
			),
		).Layout(gtx)
		elements.DrainPortals(gtx)
		nodes = elements.EndSemantics()
		r.Frame(&ops)
	}
	frame()
	m.OpenAt(f32.Pt(300, 200))
	frame()
	frame()
	if list.Size != image.Pt(180, 120) {
		t.Errorf("the menu's sibling is %v with the menu open, want 180x120", list.Size)
	}
	var rename image.Rectangle
	for _, n := range nodes {
		if n.Label == "Rename" {
			rename = n.Bounds
		}
	}
	if rename.Empty() || rename.Min.Y < 200 || rename.Min.Y > 215 || rename.Min.X < 300 || rename.Min.X > 315 {
		t.Errorf("first item at %v, want just inside (300, 200)", rename)
	}
	// A press in the title bar, outside everything the menu was laid
	// out in, closes it.
	r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: f32.Pt(500, 10), Buttons: pointer.ButtonPrimary})
	frame()
	if m.Opened() {
		t.Error("a press in the title bar left the menu open")
	}
}
