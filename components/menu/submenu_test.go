package menu

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
)

// countingIcon counts the layouts done with input enabled, which are
// the painted ones: the menu's measuring passes run with input
// disabled.
type countingIcon struct{ painted *int }

func (c countingIcon) Icon(...styles.Styles) elements.Element {
	return elements.Raw(func(gtx layout.Context) layout.Dimensions {
		if gtx.Source.Enabled() {
			*c.painted++
		}
		return layout.Dimensions{Size: image.Pt(16, 16)}
	})
}

// TestSubmenuPaintedOnce guards against an open submenu being drawn
// more than once per frame (extra copies came from the measuring
// passes, anchored at the wrong position).
func TestSubmenuPaintedOnce(t *testing.T) {
	var painted int
	sub := New().IconItem(countingIcon{&painted}, "Nested", nil)
	m := New().Item("First", nil).SubMenu("More", New().SubMenu("Deeper", sub))
	m.OpenAt(f32.Pt(100, 100))
	m.openSubmenu = 1
	m.items[1].submenu.openSubmenu = 0

	var r input.Router
	win := image.Pt(800, 600)
	elements.SetWindowSize(win)
	for frame := range 3 {
		painted = 0
		ops := new(op.Ops)
		gtx := layout.Context{Ops: ops, Source: r.Source(), Constraints: layout.Exact(win)}
		m.Layout(gtx)
		elements.DrainPortals(gtx)
		r.Frame(ops)
		if painted != 1 {
			t.Fatalf("frame %d: nested submenu painted %d times, want 1", frame, painted)
		}
	}
}

// TestSubmenuBesideItsRow checks that a submenu opens level with the row
// that triggers it, not with the top of the parent menu: moving the
// trigger row down the menu must move the submenu down by as much.
func TestSubmenuBesideItsRow(t *testing.T) {
	openAt := func(before int) (sub *Menu, rowH int) {
		sub = New().Item("Nested", nil)
		m := New()
		for range before {
			m.Item("Row", nil)
		}
		m.SubMenu("More", sub)
		m.OpenAt(f32.Pt(100, 100))
		m.openSubmenu = before

		var r input.Router
		win := image.Pt(800, 600)
		elements.SetWindowSize(win)
		ops := new(op.Ops)
		gtx := layout.Context{Ops: ops, Source: r.Source(), Constraints: layout.Exact(win)}
		m.Layout(gtx)
		elements.DrainPortals(gtx)
		if before > 0 { // items[0] is a plain row; measuring the submenu row would reopen it
			gtx.Constraints.Min = image.Point{}
			rowH = m.row(&m.items[0], 0, 0, styles.Styles{}).Layout(gtx).Size.Y
		}
		return sub, rowH
	}
	top, _ := openAt(0)
	low, rowH := openAt(3)
	if rowH <= 0 {
		t.Fatal("row height is zero")
	}
	if got, want := int(low.pos.Y-top.pos.Y), 3*rowH; got != want {
		t.Fatalf("submenu three rows down opened %dpx lower than the first row's, want %d", got, want)
	}
	if low.pos.X <= 100 {
		t.Fatalf("submenu opened at x=%v, want to the right of the parent", low.pos.X)
	}
}
