package datagrid

import (
	"encoding/csv"
	"io"
	"strings"

	"github.com/xmhrnx/giode/components/menu"
)

// ExportFormat is a text format rows are exported in.
type ExportFormat int

const (
	// CSV is comma-separated values (RFC 4180): fields with commas,
	// quotes or line breaks are quoted.
	CSV ExportFormat = iota
	// TSV is tab-separated values, quoted the same way, as spreadsheets
	// paste and open it.
	TSV
)

// Export writes every row - in the grid's current sort and filter - to
// w in format f: a header line with the shown columns' titles, then a
// line per row with their text (Format, if set). Rows a Loader hasn't
// loaded are written with empty cells.
func (g *DataGrid) Export(w io.Writer, f ExportFormat) error {
	rows := make([]int, g.src.Len())
	for i := range rows {
		rows[i] = i
	}
	return g.ExportRows(w, f, rows, true)
}

// ExportRows writes rows (source indexes, as SelectedRows returns) to w
// in format f, after a header line when header is set.
func (g *DataGrid) ExportRows(w io.Writer, f ExportFormat, rows []int, header bool) error {
	cw := csv.NewWriter(w)
	if f == TSV {
		cw.Comma = '\t'
	}
	rec := make([]string, len(g.order))
	if header {
		for j, ci := range g.order {
			rec[j] = g.cols[ci].Title
		}
		if err := cw.Write(rec); err != nil {
			return err
		}
	}
	n := g.src.Len()
	for _, r := range rows {
		if r < 0 || r >= n {
			continue
		}
		for j, ci := range g.order {
			rec[j] = formatValue(g.cols[ci], g.src.Value(r, g.cols[ci].Key))
		}
		if err := cw.Write(rec); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// CopyCSV returns the selected rows - or the focused row when none is
// selected - as CSV, without a header: what Ctrl+Shift+C copies.
func (g *DataGrid) CopyCSV() string {
	var b strings.Builder
	g.ExportRows(&b, CSV, g.copyRows(), false)
	return strings.TrimSuffix(b.String(), "\n")
}

// copyRows are the rows a copy takes: the selection, or the focused
// row.
func (g *DataGrid) copyRows() []int {
	rows := g.SelectedRows()
	if src, _ := g.at(g.focus.row); len(rows) == 0 && src >= 0 && src < g.src.Len() {
		rows = []int{src}
	}
	return rows
}

// CopyItems adds "Copy" (tab-separated, for spreadsheets) and "Copy as
// CSV" items to m, copying the selected rows (or the focused one): for a
// RowMenu or the grid's Context menu.
func (g *DataGrid) CopyItems(m *menu.Menu) {
	m.Item("Copy", func() { g.copyNext = g.CopyText() }).Shortcut("Mod+C")
	m.Item("Copy as CSV", func() { g.copyNext = g.CopyCSV() }).Shortcut("Mod+Shift+C")
}
