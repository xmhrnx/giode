package elements

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

func TestMaxWidth_CapsExpandingChild(t *testing.T) {
	var pos image.Point
	var size image.Point
	probe := Raw(func(gtx layout.Context) layout.Dimensions {
		TrackPosition(&pos)
		return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, 10)}
	})
	column := Box(styles.Styles{Width: -1, MaxWidth: 760}, probe)
	measured := Raw(func(gtx layout.Context) layout.Dimensions {
		d := column.Layout(gtx)
		size = d.Size
		return d
	})
	page := Box(styles.Styles{Width: -1, Justify: properties.JustifyCenter}, measured)

	BeginFrame()
	var ops op.Ops
	page.Layout(layout.Context{Ops: &ops, Constraints: layout.Constraints{Max: image.Pt(1200, 800)}})
	if size.X != 760 {
		t.Errorf("column width %d, want 760", size.X)
	}
	if pos.X != 220 {
		t.Errorf("column at x=%d, want 220 (centered)", pos.X)
	}
}

func TestMaxWidth_FlexGrowAndMinWins(t *testing.T) {
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: layout.Constraints{Max: image.Pt(1200, 800)}}
	// A flex parent's Flexed hands a growing child minimum = maximum =
	// its share.
	child := Box(styles.Styles{FlexGrow: 1, MaxWidth: 300}, Text("x"))
	fgtx := gtx
	fgtx.Constraints.Min.X = 1200
	grown := child.Layout(fgtx).Size
	if grown.X != 300 {
		t.Errorf("grown width %d, want 300", grown.X)
	}
	if w := Box(styles.Styles{Width: -1, MaxWidth: 100, MinWidth: 200}).Layout(gtx).Size.X; w != 200 {
		t.Errorf("min/max conflict width %d, want 200 (min wins)", w)
	}
	if h := Box(styles.Styles{Height: -1, MaxHeight: 50}).Layout(gtx).Size.Y; h != 50 {
		t.Errorf("height %d, want 50", h)
	}
}
