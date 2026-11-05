package table

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op"

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

// MeasureColumnWidths lays each row's cells out on a scratch context
// to find the widest cell per column, the same technique Menu's own
// panel-width measurement uses.
func (t *tableElement) MeasureColumnWidths(gtx layout.Context) []int {
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
