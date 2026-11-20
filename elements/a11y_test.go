package elements

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/styles"
)

func TestSemanticGroupsAndAccessible(t *testing.T) {
	view := AccessibleGroup(SemanticNode{Kind: SemanticDialog, Label: "Delete?"},
		Stack(styles.Styles{},
			Accessible(Text("Delete project", styles.Styles{}), func(n *SemanticNode) {
				n.Kind, n.Level = SemanticHeading, 2
			}),
			AccessibleGroup(SemanticNode{Kind: SemanticList}, Stack(styles.Styles{}, Text("a", styles.Styles{}), Text("b", styles.Styles{}))),
		))
	BeginFrame()
	BeginSemantics()
	var ops op.Ops
	view.Layout(layout.Context{Ops: &ops, Constraints: layout.Constraints{Max: image.Pt(400, 400)}})
	nodes := EndSemantics()
	EndFrame()
	byLabel := map[string]SemanticNode{}
	var dialog, list SemanticNode
	for _, n := range nodes {
		byLabel[n.Label] = n
		switch n.Kind {
		case SemanticDialog:
			dialog = n
		case SemanticList:
			list = n
		}
	}
	if h := byLabel["Delete project"]; h.Kind != SemanticHeading || h.Level != 2 || h.Parent != dialog.ID {
		t.Errorf("heading %+v, dialog %q", h, dialog.ID)
	}
	if a := byLabel["a"]; a.Parent != list.ID || list.Parent != dialog.ID || dialog.Parent != "" {
		t.Errorf("item parent %q, list %+v, dialog %+v", a.Parent, list, dialog)
	}
	if dialog.Bounds.Empty() || dialog.Label != "Delete?" {
		t.Errorf("dialog %+v", dialog)
	}
}
