package elements

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// TableCell wraps content as one cell: st.Padding defaults to a
// comfortable 12x8 if unset.
func TableCell(st styles.Styles, content Element) *BoxElement {
	cst := styles.Merge(styles.Styles{Padding: properties.SymmetricInset(12, 8)}, st)
	return Box(cst, content)
}

// tableRowElement is a row of cells. Its own Layout lays the cells
// out at their natural width, uncoordinated with any other row; laid
// out as part of a Table, its cells are instead each wrapped to that
// column's shared width across every row - see Table.
type tableRowElement struct {
	st    styles.Styles
	cells []Element
}

// TableRow returns one row of cells for use inside Table (or
// standalone, where its cells just take their natural width).
func TableRow(st styles.Styles, cells ...Element) *tableRowElement {
	return &tableRowElement{st: st, cells: cells}
}

// TableHeader is TableRow with a bold weight and a subtle background
// by default, to set it apart as a table's header row.
func TableHeader(st styles.Styles, cells ...Element) *tableRowElement {
	hst := styles.Merge(styles.Styles{FontWeight: properties.FontWeightBold}, st)
	return &tableRowElement{st: hst, cells: cells}
}

// Layout lays the row out at its cells' natural widths.
func (r *tableRowElement) Layout(gtx layout.Context) layout.Dimensions {
	return r.layout(gtx, nil)
}

// layout lays the row out, wrapping each cell to widths[i] when
// widths is given (Table's shared-column-width pass).
func (r *tableRowElement) layout(gtx layout.Context, widths []int) layout.Dimensions {
	rst := r.st
	rst.FlexDirection = properties.FlexDirectionRow

	cells := r.cells
	if widths != nil {
		cells = make([]Element, len(r.cells))
		for i, c := range r.cells {
			if i < len(widths) {
				c = SizedPx(widths[i], 0, c) // measured widths are pixels
			}
			cells[i] = c
		}
	}
	return Box(rst, cells...).Layout(gtx)
}

// tableElement is a stack of rows with column widths aligned across
// all of them: each row's Nth cell shares a width with every other
// row's Nth cell, the widest among them.
type tableElement struct {
	st   styles.Styles
	rows []Element
}

// Table returns a stack of rows (TableRow/TableHeader) with column
// widths aligned across all of them. A row built some other way (e.g.
// a plain Divider between sections) is passed through unmeasured.
func Table(st styles.Styles, rows ...Element) *tableElement {
	return &tableElement{st: st, rows: rows}
}

// Layout measures every row's cells to find each column's shared
// width, then lays the rows out with their cells wrapped to it.
func (t *tableElement) Layout(gtx layout.Context) layout.Dimensions {
	widths := t.measureColumnWidths(gtx)

	children := make([]Element, len(t.rows))
	for i, r := range t.rows {
		row := r
		if tr, ok := r.(*tableRowElement); ok {
			row = Raw(func(gtx layout.Context) layout.Dimensions {
				return tr.layout(gtx, widths)
			})
		}
		children[i] = row
	}

	tst := t.st
	tst.FlexDirection = properties.FlexDirectionColumn
	return Box(tst, children...).Layout(gtx)
}

// measureColumnWidths lays each row's cells out on a scratch context
// to find the widest cell per column, the same technique Menu's own
// panel-width measurement uses.
func (t *tableElement) measureColumnWidths(gtx layout.Context) []int {
	scratch := gtx
	scratch.Ops = new(op.Ops)
	scratch.Source = gtx.Source.Disabled()
	// Cleared for the same reason Menu's own panel-width measurement
	// clears it: an inherited nonzero Min would force every cell to
	// hug to at least that size, rather than its own natural width.
	scratch.Constraints.Min = image.Point{}

	var widths []int
	for _, r := range t.rows {
		tr, ok := r.(*tableRowElement)
		if !ok {
			continue
		}
		for i, cell := range tr.cells {
			dims := cell.Layout(scratch)
			for len(widths) <= i {
				widths = append(widths, 0)
			}
			if dims.Size.X > widths[i] {
				widths[i] = dims.Size.X
			}
		}
	}
	return widths
}
