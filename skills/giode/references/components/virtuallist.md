# VirtualList

A vertical list of equal-height rows that only builds the rows in or near the viewport, so a
list of a million items costs the same per frame as one of twenty.

```go
list := giode.VirtualList(
	func() int { return len(model.rows) }, // count, read every frame
	32,                                     // row height in dp
	func(i int) giode.Element { return rowView(model.rows[i]) },
	giode.Styles{Background: "theme.surface"},
).
	Overscan(5).
	OnReachEnd(loadMore).
	OnActivate(func(i int) { open(model.rows[i]) })
```

The row height is in dp like every other size; the list converts it with the display scale
itself. Each row gets exactly that height and the list's width. The list fills the space it
is given (give it a height, or `FlexGrow`).

## Methods

| Method | Effect |
| --- | --- |
| `ScrollToIndex(i, align)` | `ScrollAlignNearest` (scroll as little as needed), `ScrollAlignStart`, `ScrollAlignCenter`, `ScrollAlignEnd`. Applied on the next frame. |
| `ScrollBy(dx, dy float32)` / `ScrollOffset() (x, y float32)` | Scroll by pixels (positive `dy` scrolls down; `dx` is ignored), and read the offset. The MCP server's `ui_scroll` uses them. |
| `VisibleRange()` | First and last row at least partly visible in the last frame. |
| `Overscan(n)` | Rows built beyond each edge (default 3). |
| `OnReachEnd(fn)` | Runs once when the last row scrolls into view — for loading more. |
| `FocusedIndex()`, `SetFocusedIndex(i)` | The keyboard-focused row (-1 for none). |
| `OnFocusChange(fn)`, `OnActivate(fn)` | Focus moves; Enter on the focused row. |
| `OnKey(fn, keys...)` | Extra keys handled while focused (TreeView uses it for Left/Right). |
| `ClickFocusesRow(false)` | A press focuses the list but leaves the row to handle its own click. |
| `Focus()` | Requests keyboard focus. |
| `HasFocus()`, `OffsetDp()` | Whether the list has keyboard focus; scroll position in dp. |

## Keyboard

Click the list or Tab to it, then: Up/Down move one row, PageUp/PageDown a page, Home/End to
the ends, Enter activates. The focused row is highlighted (`BorderColor` sets the color) and
kept in view.

## Per-row state

Rows are only built while visible, so stateful widgets inside them should be looked up by
the item's key:

```go
func(i int) giode.Element {
	f := files[i]
	return giode.Keyed("files", f.ID, func() *giode.PressableElement {
		return giode.Pressable(func(s giode.PressState) giode.Element { return fileRow(f, s) })
	})
}
```

`Scroll` also gained dp helpers for lists of varying heights: `VisibleDp(top, height)`,
`OffsetDp()` and `ScrollToDp(v)`.
