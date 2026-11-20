package datagrid

import (
	"image"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/xmhrnx/giode/components/input"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// ColumnFilter is a filter on one column: the rows whose text in the
// column contains Text, ignoring case.
type ColumnFilter struct {
	Key  string `json:"key"`
	Text string `json:"text"`
}

// Filterer filters its rows: the grid calls SetFilters when the
// filters change - typed in the filter row, or set with SetFilter - so a
// server-side source can query again. SliceSource implements it in
// memory.
type Filterer interface {
	SetFilters(filters []ColumnFilter)
}

// filterDelay is how long typing in the filter row pauses before the
// filter applies.
const filterDelay = 250 * time.Millisecond

// FilterRow shows a row of text fields under the header, one for each
// Filterable column, for filtering as the user types. It returns g for
// chaining.
func (g *DataGrid) FilterRow(on bool) *DataGrid {
	g.filterRow = on
	return g
}

// OnFilter sets the handler called when the filters change, with all of
// them: for a source that isn't a Filterer, to filter it and Refresh,
// or to save them. It returns g for chaining.
func (g *DataGrid) OnFilter(fn func(filters []ColumnFilter)) *DataGrid {
	g.onFilter = fn
	return g
}

// SetFilter sets column key's filter text ("" clears it) and applies
// the filters now.
func (g *DataGrid) SetFilter(key, text string) {
	g.setFilterText(key, text)
	if in := g.filterIn[key]; in != nil && in.Text() != text {
		in.SetText(text)
	}
	g.applyFilters()
}

// ClearFilters clears every filter.
func (g *DataGrid) ClearFilters() {
	for k := range g.filters {
		if in := g.filterIn[k]; in != nil {
			in.SetText("")
		}
	}
	clear(g.filters)
	g.applyFilters()
}

// Filters returns the filters set, in column order.
func (g *DataGrid) Filters() []ColumnFilter {
	var out []ColumnFilter
	for _, ci := range g.seq {
		k := g.cols[ci].Key
		if t := g.filters[k]; t != "" {
			out = append(out, ColumnFilter{Key: k, Text: t})
		}
	}
	return out
}

func (g *DataGrid) setFilterText(key, text string) {
	if g.filters == nil {
		g.filters = map[string]string{}
	}
	if text == "" {
		delete(g.filters, key)
	} else {
		g.filters[key] = text
	}
}

// applyFilters hands the filters to the source and OnFilter, and
// rebuilds the view.
func (g *DataGrid) applyFilters() {
	g.filterDue = time.Time{}
	fs := g.Filters()
	if f, ok := g.src.(Filterer); ok {
		f.SetFilters(fs)
	}
	if g.onFilter != nil {
		g.onFilter(fs)
	}
	g.Refresh()
	if n := g.count(); g.focus.row >= n {
		g.focus.row = max(n-1, 0)
	}
	g.offY = 0
	g.changedLayout()
}

// filterInput returns column key's filter field, made on first use.
func (g *DataGrid) filterInput(key string) *input.Input {
	if g.filterIn == nil {
		g.filterIn = map[string]*input.Input{}
	}
	in := g.filterIn[key]
	if in == nil {
		in = input.New("Filter", styles.Styles{
			FontSize: g.fontSize() - 1, Padding: properties.SymmetricInset(6, 0),
			BorderWidth: 1, BorderColor: "theme.border", BorderRadius: 4, Background: "theme.background",
		})
		in.SetText(g.filters[key])
		in.OnChange(func(text string) {
			g.setFilterText(key, text)
			g.filterDue = g.frameNow.Add(filterDelay)
		})
		g.filterIn[key] = in
	}
	return in
}

// filtersDue applies filters typed long enough ago, or asks for a frame
// when they will be.
func (g *DataGrid) filtersDue(gtx layout.Context) {
	g.frameNow = gtx.Now
	if g.filterDue.IsZero() {
		return
	}
	if !gtx.Now.Before(g.filterDue) {
		g.applyFilters()
		kit.MarkLive()
		return
	}
	kit.RequestFrameAt(gtx, g.filterDue)
}

// layoutFilters lays the filter fields out in the filter row, under the
// titles, scrolling with their columns.
func (g *DataGrid) layoutFilters(gtx layout.Context, pal palette) {
	l := &g.lay
	if l.filterH == 0 {
		return
	}
	row := image.Rect(0, l.titleH, l.size.X, l.headerH)
	paint.FillShape(gtx.Ops, pal.header, clip.Rect(row).Op())
	inset := max(gtx.Dp(3), 1)
	for pass := 0; pass < 2; pass++ {
		frozen := pass == 1
		region := image.Rect(l.frozenW, l.titleH, l.size.X, l.headerH)
		c0, c1 := l.frozenCols, len(g.order)
		if frozen {
			region = image.Rect(0, l.titleH, l.frozenW, l.headerH)
			c0, c1 = 0, l.frozenCols
			paint.FillShape(gtx.Ops, pal.header, clip.Rect(region).Op())
		}
		if region.Empty() {
			continue
		}
		cl := clip.Rect(region).Push(gtx.Ops)
		for c := c0; c < c1; c++ {
			col := g.cols[g.order[c]]
			if !col.Filterable {
				continue
			}
			x := l.cellX(c, g.offX)
			w := l.colX[c+1] - l.colX[c]
			if x+w <= region.Min.X || x >= region.Max.X {
				continue
			}
			r := image.Rect(x+inset, l.titleH+inset, x+w-inset, l.headerH-inset)
			if r.Dx() <= 0 || r.Dy() <= 0 {
				continue
			}
			cgtx := gtx
			cgtx.Constraints = layout.Exact(r.Size())
			off := op.Offset(r.Min).Push(gtx.Ops)
			pop := elements.PushOffset(r.Min)
			g.filterInput(col.Key).Layout(cgtx)
			pop()
			off.Pop()
		}
		cl.Pop()
	}
}

// filtered reports whether column key has a filter, for its title to
// show it.
func (g *DataGrid) filtered(key string) bool {
	return g.filters[key] != ""
}
