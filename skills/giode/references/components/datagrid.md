# DataGrid

A table for large data sets: pipelines, contact lists, reports, issue lists. Only the rows
and columns on screen are laid out, so 100 000 rows cost what a screenful does.

```go
import "github.com/xmhrnx/giode/components/datagrid"

src := datagrid.NewSliceSource(deals, func(d Deal, col string) any {
	switch col {
	case "name":
		return d.Name
	case "owner":
		return d.Owner
	case "value":
		return d.Value
	case "closes":
		return d.Closes // a time.Time
	}
	return nil
})
grid := giode.DataGrid(src, []giode.DataGridColumn{
	{Key: "name", Title: "Deal", Width: 240, Sortable: true},
	{Key: "owner", Title: "Owner", Sortable: true},
	{Key: "value", Title: "Value", Align: giode.TextAlignEnd, Sortable: true,
		Format: func(v any) string { return fmt.Sprintf("$%.0f", v) }},
	{Key: "closes", Title: "Closes", Sortable: true},
}).Freeze(1).Checkboxes(true).OnActivate(func(row int) { openDeal(src.Row(row)) })

// In the view: the grid fills the space it's given.
giode.Stack(giode.Styles{Width: -1, Height: -1}, toolbar, grid)
```

- **Header and frozen columns.** The header stays at the top. `Freeze(n)` keeps the first
  `n` columns in place while the rest scroll sideways under them.
- **Columns.** A column has a `Key` (what the source's `Value` is asked for), a `Title`,
  a `Width` in dp (140 by default) and an `Align`. Drag a header's right edge to resize
  it, down to `MinWidth` (48 by default). `Hidden` leaves the column out. `SetColumns`
  replaces the columns, keeping the user's widths, and `Columns()` returns them with
  their current widths, to save.
- **Cells.** A value shows as text in one line, cut off with "…". Numbers, strings,
  `time.Time` and booleans show as usual, and nil as nothing. `Format` turns a value into
  text; `Cell` draws an element instead (a badge, an avatar, a progress bar), clipped to
  the cell.
- **Sorting.** Clicking a `Sortable` column's header sorts by it, again reverses it, and a
  third time clears the sort. Shift+click adds the column as another level, with numbers
  in the header showing the order. The grid asks the source to sort (`Sorter`). `OnSort`
  reports the keys, to save the sort or to sort on a server, and `Sort(keys...)` sets it
  from code.
- **Selection.**
  - A click selects a row, Shift+click a range, Ctrl/Cmd+click toggles a row.
    `Checkboxes(true)` adds a checkbox column, with one in the header for all rows.
  - Ctrl+A or the header checkbox selects everything without listing it:
    `AllSelected()` is then true, and `Unselected()` returns the exceptions. That works
    for a server's rows not yet loaded.
  - `SelectedRows`, `SelectedCount`, `Selected(row)`, `Select(row)` and `SelectAll`
    read and set the selection; `OnSelect` reports changes.
  - With a `Keyer` source (a `SliceSource` is one), the selection follows its rows
    through sorting and filtering.
- **Keyboard.**
  - Arrows move the focused cell, selecting its row; Shift extends the selection.
  - Home and End go to the first and last column, or with Ctrl/Cmd to the first and last
    row. Page Up and Page Down move a screenful.
  - Space toggles the row; Enter calls `OnActivate`, as a double click does; Ctrl/Cmd+A
    selects all.
  - Ctrl/Cmd+C copies the selected rows (or the focused one) as tab-separated text
    (`CopyText`), for pasting into a spreadsheet; Ctrl/Cmd+Shift+C copies them as CSV
    (`CopyCSV`).
- **Row menu.** `RowMenu(func(row int, col string, m *giode.MenuElement))` fills the menu a
  right-click or long-press on a row opens; `col` is the column under the pointer, for
  cell actions. `grid.CopyItems(m)` adds "Copy" and "Copy as CSV" to it.

## Filtering

`FilterRow(true)` shows a row of text fields under the header, one for each column with
`Filterable: true`. Typing filters the rows a quarter of a second after the last key: the
rows whose text in the column contains what's typed, ignoring case. A filtered column's
title shows in the accent color.

```go
grid := giode.DataGrid(src, []giode.DataGridColumn{
	{Key: "name", Title: "Name", Filterable: true, Sortable: true},
	{Key: "city", Title: "City", Filterable: true},
}).FilterRow(true)
```

The filters go to the source when it's a `datagrid.Filterer` (`SetFilters(filters
[]ColumnFilter)`): `SliceSource` filters in memory, and a server-side source sends them
with its query and reloads. `OnFilter(func([]ColumnFilter))` reports every change, for a
source that isn't a `Filterer` (filter it, then `Refresh`). `SetFilter(key, text)`,
`ClearFilters()` and `Filters()` work without the filter row too, for filters of your own
UI. The filters are part of the saved layout (`SaveLayout`).

## Export

`Export(w, datagrid.CSV)` writes every row, in the grid's sort and filter, as CSV with a
header line of the shown columns' titles; `datagrid.TSV` writes tab-separated values.
Cells are written as they show (`Format`). `ExportRows(w, format, rows, header)` writes
chosen rows, such as `SelectedRows()`. Rows a `Loader` hasn't loaded are written empty, so
export a server-side grid from the server.

```go
f, _ := os.Create(path)
defer f.Close()
grid.Export(f, datagrid.CSV)
```

## Sources

`datagrid.Source` is two methods, `Len()` and `Value(row, col)`. Optional interfaces add
the rest:

| Interface | Methods | For |
| --- | --- | --- |
| `Keyer` | `Key(row) any` | A row's identity, so the selection follows it. |
| `Sorter` | `Sort(keys []SortKey)` | Sorting when a header is clicked. |
| `Loader` | `Loaded(row) bool`, `Load(from, to int)`, `Status() (loading bool, err error)` | Server-side paging: rows not loaded show placeholders. The grid asks for the rows on screen and a page around them, shows a bar while loading, and shows an error over the rows. |
| `Filterer` | `SetFilters(filters []ColumnFilter)` | Filtering by column, in memory or on a server (see Filtering). |

`datagrid.NewSliceSource(rows, value)` is a source over a slice. It sorts in memory with
`datagrid.Compare`: numbers by value, times by time, and text case-insensitively with
numbers in it by value ("item 2" before "item 10"). It's a `Filterer`, and `Filter(keep)`
hides rows by a predicate of your own besides. `SetRows` replaces the rows, and `Row(i)`
returns the item at a row.

## Performance

At 100 000 rows, a frame of smooth scrolling takes about 0.15 ms of layout on a desktop
CPU, and a jump to an unseen part about 1.5 ms (`BenchmarkScroll100k`). It hasn't been
measured on a phone yet.

## Editing

A column with an `Editor` is editable once `OnEdit` is set:

```go
grid.OnEdit(func(row int, col string, v any) error {
	d := src.Row(row)
	switch col {
	case "value":
		if v == nil {
			return errors.New("Enter a value")
		}
		d.Value = v.(float64)
	case "stage":
		d.Stage = v.(string)
	case "won":
		d.Won = v.(bool)
	}
	return store.Save(d) // an error shows under the cell
})
```

| `Editor` | Starts with | `OnEdit` gets |
| --- | --- | --- |
| `EditText` | a field over the cell | `string` |
| `EditNumber` | a field; commas are ignored, an empty field is `nil` | `float64` or `nil` |
| `EditDate` | a field, `YYYY-MM-DD` | `time.Time` (zero when empty) |
| `EditSelect` | a menu of the column's `Options` under the cell | `string` |
| `EditCheckbox` | a click on the cell, or Space; drawn as a checkbox | `bool` |

- **Starting.** A double click on an editable cell edits it; elsewhere a double click
  calls `OnActivate`. Enter or F2 edit the focused cell.
- **Saving.** Enter saves. Tab saves and edits the next editable cell, Shift+Tab the one
  before, and Esc cancels.
- **Errors.** A value that doesn't parse (a word for a number), or an error from
  `OnEdit`, keeps the field open with the message under the cell.
- **After saving.** Change the data in `OnEdit`; the grid refreshes its groups and
  aggregates after a save.

## Grouping and aggregates

`GroupBy("stage")` shows the rows under a header row per value of the column. The headers
are in the order of their first rows under the current sort. Each header shows the value,
the number of rows, and the columns' aggregates for its rows. A click, Enter or Space on a
header collapses it; `SetGroupOpen(value, open)` does it from code.

A column's `Aggregate` summarizes its values: `datagrid.Sum`, `Avg`, `Count`, `Min`, `Max`,
or your own `func(values []any) any`. With any aggregate, a footer row shows them for all
rows. Call `Refresh()` after changing the source's rows while grouped or with aggregates.
Grouping reads every row's value, so it's for sources without a `Loader`.

## Columns menu, reordering and saved layouts

- **Header menu.** A right-click on a header opens a menu with:
  - Sort Ascending, Sort Descending and Clear Sort;
  - Group by the column, or Ungroup;
  - Hide the column;
  - a Columns submenu of every column, with a check to show or hide each one.
- **Reordering.** Dragging a header moves the column, among the frozen columns or among
  the others.
- **From code.** `SetColumnHidden(key, hidden)` and `MoveColumn(key, to)` do the same.
- **Saving the layout.** `SaveLayout()` returns the column order, widths and visibility,
  the sort and the grouping as a `GridLayout`, which encodes as JSON. `RestoreLayout`
  applies one: columns it doesn't name keep their places after those it does, and names of
  columns that are gone are skipped. `OnLayoutChange` reports every change the user makes,
  to save it.

## Not yet

Right-to-left layout is planned.

## Styles

| Property | Role |
| --- | --- |
| `FontSize` | Cell and header text, 13 by default. |
| `Background` | The rows (`theme.background`). |
| `Color` | Cell text (`theme.text`). |
| `BorderColor` | Grid lines (`theme.border`). |
| `ScrollBar` | The scrollbars. |

`RowHeight(dp)` sets the rows' height, 32 by default.
