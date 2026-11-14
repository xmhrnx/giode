package titlebar

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/components/menu"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/styles"
)

// A menu bar in the title bar, its panels laid out elsewhere in the
// view: moving across the buttons opens each panel under its own
// button. The bar measures its buttons, and a measure used to leave
// their tracked positions at the measuring pass's offset.
func TestMenuBarPanelsLaidOutElsewhere(t *testing.T) {
	fonts.UseGofont()
	elements.SetWindowSize(image.Pt(800, 600))
	defer elements.SetWindowSize(image.Point{})
	labels := []string{"New", "Undo", "Zoom"}
	menus := make([]*menu.Menu, len(labels))
	bar := menu.NewBar()
	var buttons []elements.Element
	for i, l := range labels {
		menus[i] = menu.New().Item(l, func() {})
		buttons = append(buttons, bar.Add(menus[i], elements.Box(styles.Styles{Width: 50, Height: 24})))
	}
	tb := New(&fakeWindow{focused: true}, "Doc").Leading(buttons...)
	var ops op.Ops
	var router input.Router
	gtx := layout.Context{Ops: &ops, Source: router.Source(), Constraints: layout.Exact(image.Pt(800, 600))}
	var nodes []elements.SemanticNode
	frame := func() {
		ops.Reset()
		elements.BeginSemantics()
		elements.Stack(styles.Styles{Width: -1, Height: -1}, tb,
			elements.Raw(func(gtx layout.Context) layout.Dimensions {
				for _, m := range menus {
					m.Layout(gtx)
				}
				return layout.Dimensions{}
			})).Layout(gtx)
		elements.DrainPortals(gtx)
		nodes = elements.EndSemantics()
		router.Frame(&ops)
	}
	bounds := func(label string) (button, row image.Rectangle) {
		var rows []image.Rectangle
		for _, n := range nodes {
			if n.Label == label {
				rows = append(rows, n.Bounds)
			}
		}
		if len(rows) > 0 {
			row = rows[0]
		}
		return
	}
	frame()
	frame()
	// The buttons, from the title bar's semantic tree: the Box elements
	// have none, so find them by the gaps after the title.
	var title image.Rectangle
	for _, n := range nodes {
		if n.Label == "Doc" && n.Bounds.Dx() < 100 {
			title = n.Bounds
		}
	}
	x0 := title.Max.X + 8 + 8 // the title's end padding, then the row gap
	press := func(x float32) {
		p := f32.Pt(x, 18)
		router.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: p, Buttons: pointer.ButtonPrimary},
			pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: p})
		frame()
		frame()
	}
	move := func(x float32) {
		router.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: f32.Pt(x, 18)})
		frame()
		frame()
	}
	press(float32(x0 + 25))
	if !menus[0].Opened() {
		t.Fatalf("clicking the first button at x=%d didn't open its menu", x0+25)
	}
	for i := range menus {
		left := x0 + i*58 // 50 wide, 8 apart
		move(float32(left + 25))
		if !menus[i].Opened() {
			t.Fatalf("moving onto button %d didn't open its menu", i)
		}
		_, row := bounds(labels[i])
		if row.Min.X < left || row.Min.X > left+40 || row.Min.Y < 30 {
			t.Errorf("menu %d's %q at %v, want under its button at x=%d", i, labels[i], row, left)
		}
	}
}
