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
  - Ctrl/Cmd+C copies the selected rows as tab-separated text (`CopyText`), for pasting
    into a spreadsheet.
- **Row menu.** `RowMenu(func(row int, col string, m *giode.MenuElement))` fills the menu a
  right-click or long-press on a row opens.

## Sources

`datagrid.Source` is two methods, `Len()` and `Value(row, col)`. Optional interfaces add
the rest:

| Interface | Methods | For |
| --- | --- | --- |
| `Keyer` | `Key(row) any` | A row's identity, so the selection follows it. |
| `Sorter` | `Sort(keys []SortKey)` | Sorting when a header is clicked. |
| `Loader` | `Loaded(row) bool`, `Load(from, to int)`, `Status() (loading bool, err error)` | Server-side paging: rows not loaded show placeholders. The grid asks for the rows on screen and a page around them, shows a bar while loading, and shows an error over the rows. |

`datagrid.NewSliceSource(rows, value)` is a source over a slice. It sorts in memory with
`datagrid.Compare`: numbers by value, times by time, and text case-insensitively with
numbers in it by value ("item 2" before "item 10"). `Filter(keep)` hides rows, `SetRows`
replaces them, and `Row(i)` returns the item at a row.

## Performance

At 100 000 rows, a frame of smooth scrolling takes about 0.15 ms of layout on a desktop
CPU, and a jump to an unseen part about 1.5 ms (`BenchmarkScroll100k`).

## Not yet

Inline editing, grouping with aggregate rows, column reordering and hiding from the
header, and right-to-left layout are planned.

## Styles

| Property | Role |
| --- | --- |
| `FontSize` | Cell and header text, 13 by default. |
| `Background` | The rows (`theme.background`). |
| `Color` | Cell text (`theme.text`). |
| `BorderColor` | Grid lines (`theme.border`). |
| `ScrollBar` | The scrollbars. |

`RowHeight(dp)` sets the rows' height, 32 by default.
