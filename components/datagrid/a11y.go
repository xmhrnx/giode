package datagrid

import (
	"fmt"
	"image"

	"gioui.org/layout"

	"github.com/xmhrnx/giode/elements"
)

// recordTable reports the grid to screen readers and automation: a
// table of the column headers and the rows on screen, each a row of
// cells, with its selection and place among all the rows.
func (g *DataGrid) recordTable(gtx layout.Context) {
	if !elements.SemanticsEnabled() {
		return
	}
	l := &g.lay
	origin := elements.AbsolutePosition()
	abs := func(r image.Rectangle) image.Rectangle { return r.Add(origin) }
	n := g.count()
	tbl := elements.BeginSemanticGroup(elements.SemanticNode{Kind: elements.SemanticTable, Label: g.label, Count: n})
	cellRect := func(c, y, h int) image.Rectangle {
		x := l.cellX(c, g.offX)
		return image.Rect(x, y, x+l.colX[c+1]-l.colX[c], y+h)
	}
	visible := func(r image.Rectangle) bool {
		return r.Max.X > 0 && r.Min.X < l.size.X
	}
	// The header row.
	hdr := elements.BeginSemanticGroup(elements.SemanticNode{Kind: elements.SemanticRow, Label: "Header"})
	for c, ci := range g.order {
		r := cellRect(c, 0, l.titleH)
		if !visible(r) {
			continue
		}
		col := g.cols[ci]
		desc := ""
		for i, k := range g.sort {
			if k.Key != col.Key {
				continue
			}
			desc = "sorted ascending"
			if k.Descending {
				desc = "sorted descending"
			}
			if len(g.sort) > 1 {
				desc += fmt.Sprintf(", level %d", i+1)
			}
		}
		node := elements.SemanticNode{Kind: elements.SemanticColumnHeader, Label: col.Title, Description: desc,
			Bounds: abs(r), Pos: c + 1, Count: len(g.order)}
		if col.Sortable {
			node.Target = sortActor{g, col.Key}
		}
		elements.RecordSemantic(gtx, node, r.Size())
	}
	hdr.EndAt(gtx, abs(image.Rect(0, 0, l.size.X, l.titleH)))
	// The rows on screen.
	for row := l.first; row < l.last; row++ {
		y := l.rowY(row, g.offY)
		rr := image.Rect(0, y, l.size.X, y+l.rowH)
		src, grp := g.at(row)
		if grp != nil {
			exp := elements.Expanded
			if g.collapsed[grp.value] {
				exp = elements.Collapsed
			}
			elements.RecordSemantic(gtx, elements.SemanticNode{Kind: elements.SemanticRow,
				Label: fmt.Sprintf("%s (%d)", text(grp.value), len(grp.rows)), Expanded: exp,
				Bounds: abs(rr), Pos: row + 1, Count: n, Target: groupActor{g, grp.value}}, rr.Size())
			continue
		}
		if src < 0 || src >= g.src.Len() {
			continue
		}
		rg := elements.BeginSemanticGroup(elements.SemanticNode{Kind: elements.SemanticRow, Selected: g.Selected(src),
			Focused: g.focused && g.focus.row == row, Pos: row + 1, Count: n, Target: rowActor{g, row}})
		for c, ci := range g.order {
			r := cellRect(c, y, l.rowH)
			if !visible(r) {
				continue
			}
			col := g.cols[ci]
			elements.RecordSemantic(gtx, elements.SemanticNode{Kind: elements.SemanticCell,
				Label: formatValue(col, g.src.Value(src, col.Key)), Description: col.Title, Bounds: abs(r), Pos: c + 1, Count: len(g.order)}, r.Size())
		}
		rg.EndAt(gtx, abs(rr))
	}
	tbl.EndAt(gtx, abs(image.Rectangle{Max: l.size}))
}

// rowActor activates a row for assistive technology: it selects and
// focuses it.
type rowActor struct {
	g   *DataGrid
	row int
}

func (a rowActor) Click() {
	a.g.selectOnly(a.row)
}

// sortActor sorts by a column, as a click on its header does.
type sortActor struct {
	g   *DataGrid
	key string
}

func (a sortActor) Click() {
	desc := false
	if len(a.g.sort) > 0 && a.g.sort[0].Key == a.key {
		desc = !a.g.sort[0].Descending
	}
	a.g.setSort(SortKey{Key: a.key, Descending: desc})
}

// groupActor opens or closes a group.
type groupActor struct {
	g     *DataGrid
	value any
}

func (a groupActor) Click() {
	a.g.SetGroupOpen(a.value, a.g.collapsed[a.value])
}
