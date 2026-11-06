package table

import (
	"image"

	"gioui.org/layout"

	"github.com/xmhrnx/giode/elements/box"
	"github.com/xmhrnx/giode/elements/internal/base"
	"github.com/xmhrnx/giode/elements/raw"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// tableElement is a stack of rows with column widths aligned across
// all of them: each row's Nth cell shares a width with every other
// row's Nth cell, the widest among them.
type tableElement struct {
	st   styles.Styles
	rows []base.Element
}

// Table returns a stack of rows (TableRow/TableHeader) with column
// widths aligned across all of them. A row built some other way (e.g.
// a plain Divider between sections) is passed through unmeasured.
func Table(st styles.Styles, rows ...base.Element) *tableElement {
	return &tableElement{st: st, rows: rows}
}

// Layout measures every row's cells to find each column's shared
// width, then lays the rows out with their cells wrapped to it.
func (t *tableElement) Layout(gtx layout.Context) layout.Dimensions {
	widths := t.MeasureColumnWidths(gtx)

	children := make([]base.Element, len(t.rows))
	for i, r := range t.rows {
		row := r
		if tr, ok := r.(*tableRowElement); ok {
			row = raw.Raw(func(gtx layout.Context) layout.Dimensions {
				return tr.layout(gtx, widths)
			})
		}
		children[i] = row
	}

	tst := t.st
	tst.FlexDirection = properties.FlexDirectionColumn
	return box.Box(tst, children...).Layout(gtx)
}

// MeasureColumnWidths measures each row's cells to find the widest
// cell per column. Measuring registers nothing - no selectable text or
// semantic nodes - so the cells appear once, as they're laid out.
func (t *tableElement) MeasureColumnWidths(gtx layout.Context) []int {
	scratch := gtx
	// An inherited nonzero Min would force every cell to hug to at
	// least that size, rather than its own natural width.
	scratch.Constraints.Min = image.Point{}

	var widths []int
	for _, r := range t.rows {
		tr, ok := r.(*tableRowElement)
		if !ok {
			continue
		}
		for i, cell := range tr.cells {
			dims := base.Measure(scratch, cell)
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
