package table

import (
	"gioui.org/layout"

	"github.com/xmhrnx/giode/elements/box"
	"github.com/xmhrnx/giode/elements/glyph"
	"github.com/xmhrnx/giode/elements/internal/base"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// tableRowElement is a row of cells. Its own Layout lays the cells
// out at their natural width, uncoordinated with any other row; laid
// out as part of a Table, its cells are instead each wrapped to that
// column's shared width across every row - see Table.
type tableRowElement struct {
	st    styles.Styles
	cells []base.Element
}

// TableRow returns one row of cells for use inside Table (or
// standalone, where its cells just take their natural width).
func TableRow(st styles.Styles, cells ...base.Element) *tableRowElement {
	return &tableRowElement{st: st, cells: cells}
}

// TableHeader is TableRow with a bold weight and a subtle background
// by default, to set it apart as a table's header row.
func TableHeader(st styles.Styles, cells ...base.Element) *tableRowElement {
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
		cells = make([]base.Element, len(r.cells))
		for i, c := range r.cells {
			if i < len(widths) {
				c = glyph.SizedPx(widths[i], 0, c) // measured widths are pixels
			}
			cells[i] = c
		}
	}
	return box.Box(rst, cells...).Layout(gtx)
}
