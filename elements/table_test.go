package elements

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/styles"
)

// TestTable_AlignsColumnWidthsAcrossRows checks the core promise:
// each column takes the width of its widest cell across every row,
// not just its own row - the reason Table exists over plain nested
// Rows.
func TestTable_AlignsColumnWidthsAcrossRows(t *testing.T) {
	fonts.UseGofont()

	tbl := Table(styles.Styles{},
		TableHeader(styles.Styles{}, Text("A"), Text("A very long header")),
		TableRow(styles.Styles{}, Text("row"), Text("x")),
	)

	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: layout.Constraints{Max: image.Pt(1000, 200)}}
	tbl.Layout(gtx)

	widths := tbl.measureColumnWidths(gtx)
	if len(widths) != 2 {
		t.Fatalf("measured %d columns, want 2", len(widths))
	}

	// Column 0's widest cell is "row" (row 2); column 1's is "A very
	// long header" (row 1, the header) - each should win its own
	// column regardless of which row it's in.
	scratch := gtx
	scratch.Constraints.Min = image.Point{}
	rowCellDims := Text("row").Layout(scratch)
	headerCellDims := Text("A very long header").Layout(scratch)

	if widths[0] != rowCellDims.Size.X {
		t.Errorf("column 0 width = %d, want %d (the \"row\" cell's own width)", widths[0], rowCellDims.Size.X)
	}
	if widths[1] != headerCellDims.Size.X {
		t.Errorf("column 1 width = %d, want %d (the header cell's own width)", widths[1], headerCellDims.Size.X)
	}
}

// TestTableRow_StandaloneIgnoresColumnWidths confirms a TableRow laid
// out on its own (not inside a Table) just hugs its own cells,
// unaffected by any other row.
func TestTableRow_StandaloneIgnoresColumnWidths(t *testing.T) {
	fonts.UseGofont()

	row := TableRow(styles.Styles{}, Text("x"))

	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: layout.Constraints{Max: image.Pt(1000, 200)}}
	dims := row.Layout(gtx)

	scratch := gtx
	scratch.Constraints.Min = image.Point{}
	want := Text("x").Layout(scratch)
	if dims.Size.X != want.Size.X {
		t.Errorf("standalone TableRow width = %d, want %d (its own cell's natural width)", dims.Size.X, want.Size.X)
	}
}

// TestTable_PassesThroughNonRowChildren confirms a plain element
// (e.g. a Divider between sections) mixed into Table's rows survives
// unmeasured rather than panicking or being dropped.
func TestTable_PassesThroughNonRowChildren(t *testing.T) {
	fonts.UseGofont()

	tbl := Table(styles.Styles{},
		TableRow(styles.Styles{}, Text("a")),
		Divider(styles.Styles{Height: 1}),
		TableRow(styles.Styles{}, Text("b")),
	)

	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: layout.Constraints{Max: image.Pt(300, 200)}}
	dims := tbl.Layout(gtx)
	if dims.Size.Y <= 0 {
		t.Errorf("Layout height = %d, want > 0", dims.Size.Y)
	}
}
