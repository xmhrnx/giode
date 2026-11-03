package dropdown

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/components/scroll"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// A Select far down a scrolled page: its list is sized from where the
// page shows the field, not from where the field sits in the content,
// so a long list scrolls inside the window instead of running off it.
func TestSelect_InScrolledPage(t *testing.T) {
	fonts.UseGofont()
	defer elements.SetWindowSize(image.Point{})
	sel := New(styles.Styles{}, manyOptions(40)...)
	page := scroll.New(func() elements.Element {
		return elements.Box(styles.Styles{FlexDirection: properties.FlexDirectionColumn, Width: -1},
			elements.Box(styles.Styles{Height: 2000, Width: -1}),
			sel,
			elements.Box(styles.Styles{Height: 1000, Width: -1}),
		)
	})
	var r input.Router
	win := image.Pt(400, 600)
	frame := func() {
		elements.SetWindowSize(win)
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Source: r.Source(), Constraints: layout.Constraints{Max: win}}
		page.Layout(gtx)
		elements.DrainPortals(gtx)
		r.Frame(&ops)
	}
	tap := func(x, y float32) {
		r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: f32.Pt(x, y), Buttons: pointer.ButtonPrimary},
			pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: f32.Pt(x, y)})
		frame()
		frame()
	}
	frame()
	page.ScrollTo(1900) // the field shows at y = 100
	frame()
	frame()
	tap(20, 110)
	if !sel.Opened() {
		t.Fatal("the list didn't open")
	}
	if sel.list == nil {
		t.Fatal("40 options below a field at y=100 in a 600px window didn't scroll")
	}
	if v := sel.list.ViewportHeight(); v <= 0 || v > 600-100 {
		t.Fatalf("list viewport %d, want within the room below the field", v)
	}
	before := sel.list.Offset()
	r.Queue(pointer.Event{Kind: pointer.Scroll, Source: pointer.Mouse, Position: f32.Pt(200, 400), Scroll: f32.Pt(0, 150)})
	frame()
	frame()
	if sel.list.Offset() <= before {
		t.Errorf("wheel didn't scroll the list: offset %v", sel.list.Offset())
	}
	if page.Offset() != 1900 {
		t.Errorf("the page scrolled under the list: %v", page.Offset())
	}
	// Anywhere on a row picks it, not only its text.
	tap(380, 300)
	if sel.Opened() || sel.Selected() == 0 {
		t.Errorf("tapping a row's right end: open=%v selected=%d, want a choice", sel.Opened(), sel.Selected())
	}
}
