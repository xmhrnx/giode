package datagrid

import (
	"fmt"
	"slices"
)

// Grouping: with GroupBy set, the grid shows the source's rows under a
// header row per value of a column, each header collapsible and showing
// its rows' count and aggregates. The rows the grid shows are then the
// view: group headers and the rows of open groups. Without grouping
// there's no view and display rows are source rows.

// viewRow is a row the grid shows: a source row, or (group >= 0) a
// group's header.
type viewRow struct {
	src, group int
}

// group is the rows sharing a value of the grouping column.
type group struct {
	value any
	rows  []int
}

// GroupBy groups the rows by the column with key col; "" ungroups.
// Groups follow the sort's order of their first rows, and start open.
// Call Refresh after the source's rows change. Grouping needs every
// row's value: it's for sources without a Loader.
func (g *DataGrid) GroupBy(col string) {
	g.groupBy = col
	g.collapsed = map[any]bool{}
	g.Refresh()
	g.changedLayout()
}

// GroupedBy returns the grouping column's key, or "".
func (g *DataGrid) GroupedBy() string { return g.groupBy }

// SetGroupOpen opens or closes the group of rows whose grouping value
// is v.
func (g *DataGrid) SetGroupOpen(v any, open bool) {
	if g.collapsed == nil {
		g.collapsed = map[any]bool{}
	}
	g.collapsed[v] = !open
	g.rebuildView()
}

// Refresh rebuilds the groups and aggregates from the source: call it
// after changing the source's rows while grouped or with aggregates.
func (g *DataGrid) Refresh() {
	g.rebuildView()
	g.aggs = nil
}

// rebuildView builds the groups and the view.
func (g *DataGrid) rebuildView() {
	g.groups, g.view = g.groups[:0], g.view[:0]
	if g.groupBy == "" {
		g.view = nil
		return
	}
	n := g.src.Len()
	index := map[any]int{}
	for r := range n {
		v := g.src.Value(r, g.groupBy)
		k := groupKey(v)
		i, ok := index[k]
		if !ok {
			i = len(g.groups)
			index[k] = i
			g.groups = append(g.groups, group{value: v})
		}
		g.groups[i].rows = append(g.groups[i].rows, r)
	}
	for gi, gr := range g.groups {
		g.view = append(g.view, viewRow{src: -1, group: gi})
		if g.collapsed[groupKey(gr.value)] {
			continue
		}
		for _, r := range gr.rows {
			g.view = append(g.view, viewRow{src: r, group: -1})
		}
	}
	if g.view == nil {
		g.view = []viewRow{}
	}
}

// groupKey makes a value usable as a map key.
func groupKey(v any) any {
	switch v.(type) {
	case nil, string, bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return v
	}
	return fmt.Sprint(v)
}

// count is how many rows the grid shows.
func (g *DataGrid) count() int {
	if g.view != nil {
		return len(g.view)
	}
	return g.src.Len()
}

// at returns display row i's source row, or -1 and its group for a
// group header.
func (g *DataGrid) at(i int) (src int, grp *group) {
	if g.view == nil {
		return i, nil
	}
	if i < 0 || i >= len(g.view) {
		return -1, nil
	}
	v := g.view[i]
	if v.group >= 0 {
		return -1, &g.groups[v.group]
	}
	return v.src, nil
}

// displayOf returns the display row showing source row src, or -1.
func (g *DataGrid) displayOf(src int) int {
	if g.view == nil {
		return src
	}
	return slices.IndexFunc(g.view, func(v viewRow) bool { return v.group < 0 && v.src == src })
}

// Aggregates.

// Sum adds a column's numbers.
func Sum(values []any) any {
	t := 0.0
	for _, v := range values {
		if x, ok := number(v); ok {
			t += x
		}
	}
	return t
}

// Avg averages a column's numbers.
func Avg(values []any) any {
	t, n := 0.0, 0
	for _, v := range values {
		if x, ok := number(v); ok {
			t += x
			n++
		}
	}
	if n == 0 {
		return nil
	}
	return t / float64(n)
}

// Count counts a column's values that aren't nil.
func Count(values []any) any {
	n := 0
	for _, v := range values {
		if v != nil {
			n++
		}
	}
	return n
}

// Min and Max return a column's least and greatest value, by Compare.
func Min(values []any) any { return extreme(values, -1) }
func Max(values []any) any { return extreme(values, 1) }

func extreme(values []any, sign int) any {
	var best any
	for _, v := range values {
		if v == nil {
			continue
		}
		if best == nil || Compare(v, best)*sign > 0 {
			best = v
		}
	}
	return best
}

// aggregate returns column ci's aggregate over rows (all rows when
// nil), cached for the footer until Refresh.
func (g *DataGrid) aggregate(ci int, rows []int) any {
	col := g.cols[ci]
	if col.Aggregate == nil {
		return nil
	}
	if rows == nil {
		if g.aggs == nil {
			g.aggs = map[string]any{}
		}
		if v, ok := g.aggs[col.Key]; ok {
			return v
		}
		vals := make([]any, g.src.Len())
		for r := range vals {
			vals[r] = g.src.Value(r, col.Key)
		}
		v := col.Aggregate(vals)
		g.aggs[col.Key] = v
		return v
	}
	vals := make([]any, len(rows))
	for i, r := range rows {
		vals[i] = g.src.Value(r, col.Key)
	}
	return col.Aggregate(vals)
}

// hasFooter reports whether any column has an aggregate.
func (g *DataGrid) hasFooter() bool {
	for _, ci := range g.order {
		if g.cols[ci].Aggregate != nil {
			return true
		}
	}
	return false
}
