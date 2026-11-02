package elements

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

func TestAlignStretch(t *testing.T) {
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: layout.Constraints{Max: image.Pt(600, 400)}}
	var a, b layout.Dimensions
	probe := func(d *layout.Dimensions, h properties.Height) Element {
		return Raw(func(gtx layout.Context) layout.Dimensions {
			*d = Box(styles.Styles{Width: 100, Height: h}).Layout(gtx)
			*d = layout.Dimensions{Size: image.Pt(d.Size.X, max(d.Size.Y, gtx.Constraints.Min.Y))}
			return *d
		})
	}
	row := Row(styles.Styles{Align: properties.AlignStretch, Gap: 8}, probe(&a, 40), probe(&b, 120))
	dims := row.Layout(gtx)
	if a.Size.Y != 120 || b.Size.Y != 120 || dims.Size.Y != 120 {
		t.Errorf("stretched row: children %d and %d tall, row %d; want 120 each", a.Size.Y, b.Size.Y, dims.Size.Y)
	}
	// Cards: a box sized by its content grows to the tallest.
	var inner int
	short := Box(styles.Styles{Width: 100, Background: "#fff"}, Raw(func(gtx layout.Context) layout.Dimensions {
		inner = gtx.Constraints.Max.Y
		return layout.Dimensions{Size: image.Pt(100, 30)}
	}))
	tall := Box(styles.Styles{Width: 100, Background: "#fff"}, Box(styles.Styles{Height: 90}))
	if d := Row(styles.Styles{Align: properties.AlignStretch}, short, tall).Layout(gtx); d.Size.Y != 90 {
		t.Errorf("row %d tall, want 90", d.Size.Y)
	}
	if inner != 90 {
		t.Errorf("the short card's content may be %d tall, want the stretched 90", inner)
	}
}
