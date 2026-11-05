package elements

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

func TestSemantics_FlexChildBounds(t *testing.T) {
	BeginSemantics()
	gtx := layout.Context{Ops: new(op.Ops), Constraints: layout.Constraints{Max: image.Pt(400, 400)}}
	sized := func(w, h int, label string) Element {
		return Raw(func(gtx layout.Context) layout.Dimensions {
			RecordSemantic(gtx, SemanticNode{Kind: SemanticCustom, Label: label}, image.Pt(w, h))
			return layout.Dimensions{Size: image.Pt(w, h)}
		})
	}
	Row(styles.Styles{Gap: 10, Padding: styles.Styles{}.Padding},
		sized(50, 20, "a"),
		Stack(styles.Styles{Gap: 5}, sized(30, 10, "b"), sized(30, 10, "c")),
	).Layout(gtx)
	nodes := EndSemantics()
	want := map[string]image.Point{"a": {0, 0}, "b": {60, 0}, "c": {60, 15}}
	for _, n := range nodes {
		if got := n.Bounds.Min; got != want[n.Label] {
			t.Errorf("%s at %v, want %v", n.Label, got, want[n.Label])
		}
	}
	if len(nodes) != 3 {
		t.Fatalf("nodes = %d, want 3", len(nodes))
	}
}

func TestRow_AlignCenterDoesNotStretchChildren(t *testing.T) {
	BeginSemantics()
	gtx := layout.Context{Ops: new(op.Ops), Constraints: layout.Constraints{Max: image.Pt(400, 400)}}
	child := Raw(func(gtx layout.Context) layout.Dimensions {
		size := image.Pt(20, max(10, gtx.Constraints.Min.Y))
		RecordSemantic(gtx, SemanticNode{Kind: SemanticCustom, Label: "c"}, size)
		return layout.Dimensions{Size: size}
	})
	for _, wrap := range []properties.Wrap{properties.WrapNoWrap, properties.WrapWrap} {
		BeginSemantics()
		dims := Row(styles.Styles{Height: 40, Align: properties.AlignCenter, Wrap: wrap}, child).Layout(gtx)
		n := EndSemantics()[0]
		if dims.Size.Y != 40 {
			t.Errorf("wrap %q: row height = %d, want 40", wrap, dims.Size.Y)
		}
		if n.Bounds.Dy() != 10 || n.Bounds.Min.Y != 15 {
			t.Errorf("wrap %q: child at %v, want 10 tall centered at y=15", wrap, n.Bounds)
		}
	}
	EndSemantics()
}

func TestRow_FlexGrowBoxFills(t *testing.T) {
	BeginSemantics()
	gtx := layout.Context{Ops: new(op.Ops), Constraints: layout.Constraints{Max: image.Pt(400, 100)}}
	fixed := func(w int, label string) Element {
		return Raw(func(gtx layout.Context) layout.Dimensions {
			RecordSemantic(gtx, SemanticNode{Kind: SemanticCustom, Label: label}, image.Pt(w, 10))
			return layout.Dimensions{Size: image.Pt(w, 10)}
		})
	}
	Row(styles.Styles{Width: -1, Wrap: properties.WrapNoWrap},
		Box(styles.Styles{FlexGrow: 1}, fixed(50, "grow")),
		fixed(60, "end"),
	).Layout(gtx)
	for _, n := range EndSemantics() {
		if n.Label == "end" && n.Bounds.Min.X != 340 {
			t.Errorf("item after a FlexGrow box at x=%d, want 340 (pushed to the end)", n.Bounds.Min.X)
		}
	}
}
