package elements

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/styles"
)

func TestText_ExpandsTabs(t *testing.T) {
	fonts.UseGofont()
	BeginSemantics()
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: layout.Constraints{Max: image.Pt(800, 200)}}
	Text("a\tb", styles.Styles{TabWidth: 2}).Layout(gtx)
	nodes := EndSemantics()
	if len(nodes) != 1 || nodes[0].Label != "a b" {
		t.Errorf("semantic label %v, want the expanded text %q", nodes, "a b")
	}
}
