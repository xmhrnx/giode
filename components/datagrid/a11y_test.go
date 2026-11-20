package datagrid

import (
	"image"
	"testing"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/elements"
)

func TestTableSemantics(t *testing.T) {
	g := New(contacts(), contactCols()).Label("Contacts")
	g.Sort(SortKey{Key: "name"})
	h := newHarness(g)
	_ = h
	elements.BeginFrame()
	elements.BeginSemantics()
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Now: time.Unix(2000, 0), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(600, 400))}
	g.Layout(gtx)
	nodes := elements.EndSemantics()
	elements.EndFrame()
	var table elements.SemanticNode
	rows, headers, cells := 0, 0, 0
	byID := map[string]elements.SemanticNode{}
	for _, n := range nodes {
		byID[n.ID] = n
	}
	var firstRow elements.SemanticNode
	for _, n := range nodes {
		switch n.Kind {
		case elements.SemanticTable:
			table = n
		case elements.SemanticColumnHeader:
			headers++
			if n.Label == "Name" && n.Description != "sorted ascending" {
				t.Errorf("name header %+v", n)
			}
		case elements.SemanticRow:
			if n.Label != "Header" {
				rows++
				if n.Pos == 1 {
					firstRow = n
				}
			}
		case elements.SemanticCell:
			cells++
		}
	}
	if table.Label != "Contacts" || table.Count != 4 || headers != 3 || rows != 4 || cells != 12 {
		t.Fatalf("table %+v, %d headers, %d rows, %d cells", table, headers, rows, cells)
	}
	if firstRow.Parent != table.ID || firstRow.Count != 4 {
		t.Errorf("first row %+v", firstRow)
	}
	// Activating a row selects it.
	firstRow.Target.(interface{ Click() }).Click()
	if !g.Selected(0) {
		t.Error("activating the row didn't select it")
	}
}
