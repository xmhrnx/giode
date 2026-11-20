// Package datagrid provides DataGrid: a virtualized table for large
// data sets - pipelines, contact lists, reports, issue lists - with a
// header that stays at the top, frozen leading columns, resizable
// columns, sorting on one or several columns, row selection by click,
// Shift, Ctrl/Cmd and checkboxes, keyboard navigation and copying rows
// as tab-separated text.
//
// Only the rows and columns on screen are laid out, so a grid of
// 100 000 rows costs what a screenful does.
package datagrid

import (
	"image"
	"slices"
	"time"

	"gioui.org/gesture"
	"gioui.org/layout"

	"github.com/xmhrnx/giode/components/input"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/components/menu"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Column describes one column of a grid.
type Column struct {
	// Key identifies the column to the Source (Value), in sorting and
	// in a saved layout.
	Key string
	// Title is the header's text.
	Title string
	// Width is the column's width in dp, 140 by default; the user can
	// resize it, down to MinWidth (48 by default).
	Width, MinWidth int
	// Align is the cells' alignment: TextAlignEnd for numbers.
	Align properties.TextAlign
	// Sortable lets a click on the header sort by the column.
	Sortable bool
	// Filterable gives the column a field in the filter row (FilterRow).
	Filterable bool
	// Hidden leaves the column out.
	Hidden bool
	// Format turns a value into the cell's text, for showing and
	// copying. By default numbers, strings, times and booleans show as
	// usual, and nil as nothing.
	Format func(v any) string
	// Cell draws a cell instead of its text: a badge, an avatar, a
	// progress bar. It's laid out in the cell, which clips it.
	Cell func(row int, v any) elements.Element
	// Editor makes the column's cells editable (see OnEdit): EditText,
	// EditNumber, EditDate, EditSelect from Options, EditCheckbox.
	Editor  EditorKind
	Options []string
	// Aggregate summarizes the column's values for the footer row and
	// group headers: Sum, Avg, Count, Min, Max, or your own.
	Aggregate func(values []any) any
}

// EditorKind is how a column's cells are edited.
type EditorKind uint8

const (
	// EditNone: not editable.
	EditNone EditorKind = iota
	// EditText edits the text; OnEdit gets a string.
	EditText
	// EditNumber edits a number; OnEdit gets a float64.
	EditNumber
	// EditDate edits a date as YYYY-MM-DD; OnEdit gets a time.Time.
	EditDate
	// EditSelect picks one of the column's Options from a menu;
	// OnEdit gets the string.
	EditSelect
	// EditCheckbox flips a bool with a click or Space; OnEdit gets the
	// bool.
	EditCheckbox
)

// SortKey is one level of a sort: a column's key, ascending or not.
type SortKey struct {
	Key        string
	Descending bool
}

// DataGrid is a stateful, virtualized table. Create it once with New
// and lay it out every frame:
//
//	grid := datagrid.New(source,
//		datagrid.Column{Key: "name", Title: "Name", Width: 200, Sortable: true},
//		datagrid.Column{Key: "value", Title: "Value", Align: giode.TextAlignEnd, Sortable: true},
//	).Freeze(1).Checkboxes(true).OnActivate(openDeal)
type DataGrid struct {
	// Contextual gives it a context menu (Context), over the whole
	// grid; RowMenu builds one per row.
	elements.Contextual

	src     Source
	cols    []Column
	seq     []int // every column in display order, hidden ones too
	order   []int // the shown columns in display order, by index into cols
	widths  []float32
	frozen  int
	checks  bool
	st      styles.Styles
	rowH    int // dp
	headerH int // dp

	groupBy   string
	collapsed map[any]bool
	groups    []group
	view      []viewRow
	aggs      map[string]any

	onEdit   func(row int, col string, v any) error
	onLayout func(GridLayout)
	edit     editState
	hmenu    *menu.Menu
	headCtx  elements.Contextual
	origin   image.Point
	dragCol  int // the display column being dragged to a new place, or -1
	dragFrom float32
	dragging bool

	sort       []SortKey
	onSort     func(keys []SortKey)
	onSelect   func()
	onActivate func(row int)
	rowMenu    func(row int, col string, m *menu.Menu)

	sel   selection
	focus cell
	// anchor is where a Shift selection runs from.
	anchor int

	// Scrolling, in px.
	offX, offY       float32
	scrollY, scrollX gesture.Scroll
	barY, barX       kit.Scrollbar
	gest             kit.ScrollGesture

	// The last layout, in px, for hit tests and scrolling into view.
	lay frameLayout

	// Pointer state.
	hoverRow  int
	resizing  int // the display column being resized, or -1
	resizeX   float32
	resizeW   float32
	pressRow  int
	pressCol  int
	clickedAt int64
	lastClick int
	focused   bool
	reveal    bool
	pass      func(gtx layout.Context) bool

	menu    *menu.Menu
	rowCtx  elements.Contextual
	menuRow int
	menuCol string
	// copyNext is text a menu's Copy item put on the clipboard next
	// frame.
	copyNext string
	// label names the table for screen readers (Label).
	label string

	// Filtering (see filter.go).
	filterRow bool
	filters   map[string]string
	filterIn  map[string]*input.Input
	filterDue time.Time
	// frameNow is the last frame's time, for the filter row's delay.
	frameNow time.Time
	onFilter func([]ColumnFilter)
}

// cell is a position in the grid: a row, and a column in display
// order.
type cell struct {
	row, col int
}

// New returns a grid showing src's rows in cols. The styles argument is
// optional: FontSize is the cells' text size (13 by default),
// Background the rows', BorderColor the grid lines.
func New(src Source, cols []Column, st ...styles.Styles) *DataGrid {
	g := &DataGrid{src: src, rowH: 32, headerH: 36, resizing: -1, hoverRow: -1, pressRow: -1, menuRow: -1, dragCol: -1}
	if len(st) > 0 {
		g.st = st[0]
	}
	g.SetColumns(cols)
	g.menu = menu.New()
	g.hmenu = menu.New()
	g.headCtx.Context(g.hmenu)
	return g
}

// SetColumns replaces the columns, keeping the widths the user gave
// columns that are still there.
func (g *DataGrid) SetColumns(cols []Column) {
	old := map[string]float32{}
	for i, c := range g.cols {
		if i < len(g.widths) {
			old[c.Key] = g.widths[i]
		}
	}
	g.cols = append([]Column(nil), cols...)
	g.widths = make([]float32, len(cols))
	g.seq = g.seq[:0]
	for i, c := range g.cols {
		w := float32(c.Width)
		if w <= 0 {
			w = 140
		}
		if prev, ok := old[c.Key]; ok {
			w = prev
		}
		g.widths[i] = w
		g.seq = append(g.seq, i)
	}
	g.reorder()
	g.aggs = nil
}

// reorder derives the shown columns from seq.
func (g *DataGrid) reorder() {
	g.order = g.order[:0]
	for _, ci := range g.seq {
		if !g.cols[ci].Hidden {
			g.order = append(g.order, ci)
		}
	}
}

// colIndex returns the column with key's index, or -1.
func (g *DataGrid) colIndex(key string) int {
	for i, c := range g.cols {
		if c.Key == key {
			return i
		}
	}
	return -1
}

// SetColumnHidden shows or hides the column with key, as the header
// menu does.
func (g *DataGrid) SetColumnHidden(key string, hidden bool) {
	if ci := g.colIndex(key); ci >= 0 {
		g.cols[ci].Hidden = hidden
		g.reorder()
		g.focus.col = min(g.focus.col, max(len(g.order)-1, 0))
		g.changedLayout()
	}
}

// MoveColumn puts the column with key at position to among all the
// columns, as dragging its header does.
func (g *DataGrid) MoveColumn(key string, to int) {
	ci := g.colIndex(key)
	if ci < 0 {
		return
	}
	i := slices.Index(g.seq, ci)
	g.seq = slices.Delete(g.seq, i, i+1)
	to = min(max(to, 0), len(g.seq))
	g.seq = slices.Insert(g.seq, to, ci)
	g.reorder()
	g.changedLayout()
}

// GridLayout is what the user changed about a grid - column order,
// widths and visibility, the sort, the grouping, the filters - to save (it encodes
// as JSON) and restore with RestoreLayout.
type GridLayout struct {
	Columns []ColumnLayout `json:"columns"`
	Sort    []SortKey      `json:"sort,omitempty"`
	GroupBy string         `json:"groupBy,omitempty"`
	Filters []ColumnFilter `json:"filters,omitempty"`
}

// ColumnLayout is a column's place in a GridLayout.
type ColumnLayout struct {
	Key    string `json:"key"`
	Width  int    `json:"width"`
	Hidden bool   `json:"hidden,omitempty"`
}

// SaveLayout returns the grid's layout.
func (g *DataGrid) SaveLayout() GridLayout {
	var l GridLayout
	for _, ci := range g.seq {
		c := g.cols[ci]
		l.Columns = append(l.Columns, ColumnLayout{Key: c.Key, Width: int(g.widths[ci] + 0.5), Hidden: c.Hidden})
	}
	l.Sort, l.GroupBy, l.Filters = g.SortKeys(), g.groupBy, g.Filters()
	return l
}

// RestoreLayout applies a saved layout. Columns it doesn't name keep
// their place after the ones it does; names of columns gone are
// skipped.
func (g *DataGrid) RestoreLayout(l GridLayout) {
	var seq []int
	seen := map[int]bool{}
	for _, cl := range l.Columns {
		ci := g.colIndex(cl.Key)
		if ci < 0 || seen[ci] {
			continue
		}
		seen[ci] = true
		seq = append(seq, ci)
		if cl.Width > 0 {
			g.widths[ci] = float32(cl.Width)
		}
		g.cols[ci].Hidden = cl.Hidden
	}
	for _, ci := range g.seq {
		if !seen[ci] {
			seq = append(seq, ci)
		}
	}
	g.seq = seq
	g.reorder()
	g.Sort(l.Sort...)
	g.groupBy = l.GroupBy
	g.collapsed = map[any]bool{}
	if len(l.Filters) > 0 || len(g.filters) > 0 {
		clear(g.filters)
		for _, f := range l.Filters {
			g.setFilterText(f.Key, f.Text)
			if in := g.filterIn[f.Key]; in != nil {
				in.SetText(f.Text)
			}
		}
		g.applyFilters()
		return
	}
	g.Refresh()
}

// OnLayoutChange sets the handler called when the user changes the
// layout - resizes, moves, hides or shows a column, sorts, groups - to
// save it. It returns g for chaining.
func (g *DataGrid) OnLayoutChange(fn func(GridLayout)) *DataGrid {
	g.onLayout = fn
	return g
}

func (g *DataGrid) changedLayout() {
	if g.onLayout != nil {
		g.onLayout(g.SaveLayout())
	}
}

// OnEdit sets the handler that saves an edited cell: row, the column's
// key and the new value, typed by the column's Editor. An error keeps
// the editor open and shows the error under the cell; nil closes it.
// It returns g for chaining.
func (g *DataGrid) OnEdit(fn func(row int, col string, v any) error) *DataGrid {
	g.onEdit = fn
	return g
}

// Columns returns the columns, with their current widths in dp.
func (g *DataGrid) Columns() []Column {
	out := append([]Column(nil), g.cols...)
	for i := range out {
		out[i].Width = int(g.widths[i] + 0.5)
	}
	return out
}

// Label names the table for screen readers ("Deals"). It returns g for
// chaining.
func (g *DataGrid) Label(s string) *DataGrid {
	g.label = s
	return g
}

// Freeze keeps the first n columns in place while the others scroll
// sideways. It returns g for chaining.
func (g *DataGrid) Freeze(n int) *DataGrid {
	g.frozen = max(n, 0)
	return g
}

// Checkboxes adds a leading column of checkboxes selecting rows, with
// one in the header selecting them all. It returns g for chaining.
func (g *DataGrid) Checkboxes(on bool) *DataGrid {
	g.checks = on
	return g
}

// RowHeight sets the rows' height in dp, 32 by default. It returns g
// for chaining.
func (g *DataGrid) RowHeight(dp int) *DataGrid {
	if dp > 0 {
		g.rowH = dp
	}
	return g
}

// OnSort sets the handler called when the user changes the sort, with
// its keys: a header click sorts by that column, again reverses it, a
// third time clears it; Shift+click adds the column to the sort. The
// grid asks the source to sort (Sorter); OnSort is for saving the sort
// or sorting on a server. It returns g for chaining.
func (g *DataGrid) OnSort(fn func(keys []SortKey)) *DataGrid {
	g.onSort = fn
	return g
}

// OnSelect sets the handler called when the selection changes. It
// returns g for chaining.
func (g *DataGrid) OnSelect(fn func()) *DataGrid {
	g.onSelect = fn
	return g
}

// OnActivate sets the handler for a double click or Enter on a row:
// open the deal, the contact, the issue. It returns g for chaining.
func (g *DataGrid) OnActivate(fn func(row int)) *DataGrid {
	g.onActivate = fn
	return g
}

// RowMenu lets fn fill the menu a right-click (or long-press) on a row
// opens, given the row and the column's key. It returns g for chaining.
func (g *DataGrid) RowMenu(fn func(row int, col string, m *menu.Menu)) *DataGrid {
	g.rowMenu = fn
	g.rowCtx.Context(g.menu)
	return g
}

// Sort sorts by keys, as a header click would, without calling OnSort.
func (g *DataGrid) Sort(keys ...SortKey) {
	g.sort = append([]SortKey(nil), keys...)
	if s, ok := g.src.(Sorter); ok {
		s.Sort(g.sort)
	}
	if g.groupBy != "" {
		g.rebuildView()
	}
}

// SortKeys returns the current sort.
func (g *DataGrid) SortKeys() []SortKey {
	return append([]SortKey(nil), g.sort...)
}

// Selected reports whether row is selected.
func (g *DataGrid) Selected(row int) bool {
	return g.sel.has(g.keyOf(row))
}

// SelectedRows returns the selected rows in order. With everything
// selected that's every row, so prefer AllSelected and Unselected for
// large sources.
func (g *DataGrid) SelectedRows() []int {
	var out []int
	for i := range g.src.Len() {
		if g.Selected(i) {
			out = append(out, i)
		}
	}
	return out
}

// AllSelected reports whether everything is selected (Ctrl+A, or the
// header checkbox) - across pages a server hasn't sent - but the rows
// Unselected returns.
func (g *DataGrid) AllSelected() bool { return g.sel.all }

// Unselected returns the keys (Keyer, else row indices) left out of an
// AllSelected selection.
func (g *DataGrid) Unselected() []any {
	if !g.sel.all {
		return nil
	}
	out := make([]any, 0, len(g.sel.ex))
	for k := range g.sel.ex {
		out = append(out, k)
	}
	return out
}

// SelectedCount returns how many rows are selected.
func (g *DataGrid) SelectedCount() int { return g.sel.count(g.src.Len()) }

// SelectAll selects every row, or none.
func (g *DataGrid) SelectAll(on bool) {
	g.sel = selection{all: on}
	g.selected()
}

// Select selects row alone, focuses it and scrolls it into view.
func (g *DataGrid) Select(row int) {
	if d := g.displayOf(row); d >= 0 {
		g.selectOnly(d)
	}
}

// selectOnly selects display row d alone, focusing it.
func (g *DataGrid) selectOnly(d int) {
	g.sel = selection{}
	if src, _ := g.at(d); src >= 0 {
		g.sel.set(g.keyOf(src), true)
	}
	g.focus.row, g.anchor = d, d
	g.reveal = true
	g.selected()
}

// Focused returns the row (-1 on a group header) and column key of the
// focused cell.
func (g *DataGrid) Focused() (row int, col string) {
	if g.focus.col >= 0 && g.focus.col < len(g.order) {
		col = g.cols[g.order[g.focus.col]].Key
	}
	row, _ = g.at(g.focus.row)
	return row, col
}

func (g *DataGrid) selected() {
	if g.onSelect != nil {
		g.onSelect()
	}
}

// keyOf is row's identity in the selection: the source's Key when it
// has one, so the selection survives a sort.
func (g *DataGrid) keyOf(row int) any {
	if k, ok := g.src.(Keyer); ok {
		return k.Key(row)
	}
	return row
}

// selection is the selected rows: those in ex, or with all, all but
// those in ex.
type selection struct {
	all bool
	ex  map[any]bool
}

func (s *selection) has(k any) bool { return s.all != s.ex[k] }

func (s *selection) set(k any, on bool) {
	if s.ex == nil {
		s.ex = map[any]bool{}
	}
	if on != s.all {
		s.ex[k] = true
	} else {
		delete(s.ex, k)
	}
}

func (s *selection) count(total int) int {
	if s.all {
		return total - len(s.ex)
	}
	return len(s.ex)
}
