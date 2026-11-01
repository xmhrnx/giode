# Table

Stateless rows of cells with column widths aligned across all of them: each row's Nth
cell shares a width with every other row's Nth cell, the widest among them.

```go
giode.Table(giode.Styles{},
	giode.TableHeader(giode.Styles{Background: "#334155"},
		giode.TableCell(giode.Styles{}, giode.Text("Name")),
		giode.TableCell(giode.Styles{}, giode.Text("Role")),
	),
	giode.TableRow(giode.Styles{},
		giode.TableCell(giode.Styles{}, giode.Text("Ada")),
		giode.TableCell(giode.Styles{}, giode.Text("Engineer")),
	),
)
```

Like `Box`/`Row`/`Stack`, Table and its rows/cells are stateless: build them fresh every
frame, no `New` constructor or persistent state involved.

## API

| Member | Description |
| --- | --- |
| `Table(st Styles, rows ...Element) Element` | Stacks rows, aligning column widths across all of them. |
| `TableHeader(st Styles, cells ...Element) Element` | A row with a bold weight by default, to set it apart as the header. |
| `TableRow(st Styles, cells ...Element) Element` | A row of cells. |
| `TableCell(st Styles, content Element) Element` | Wraps content as one cell; `Padding` defaults to 12x8. |

A row built some other way (e.g. a plain `Divider` mixed in between sections) passes
through `Table` unmeasured, rather than being treated as a row of cells.

## Behavior

- Column widths come from each column's widest cell across every row (measured on a
  scratch pass, the same technique Menu's own panel-width measurement uses) - not fixed
  or evenly split.
- `TableRow`/`TableHeader` built standalone (not inside a `Table`) just hug their own
  cells' natural widths, uncoordinated with any other row.
- No sorting or row selection - it's a layout primitive; build that behavior on top with
  `TableCell`'s own `Styles` (e.g. a `Background` for a selected row) and ordinary Go
  slice sorting before building the rows.
