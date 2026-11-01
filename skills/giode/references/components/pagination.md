# Pagination

Stateful row of page-number controls, built on [Button](button.md): prev/next and a
windowed set of page buttons (first, last, and a few pages around the current one, with
an ellipsis for the gap on a large page count).

```go
pager := giode.Pagination(20). // 20 total pages
	OnChange(func(page int) { loadPage(page) })

// in the view:
giode.Stack(giode.Styles{}, pager)
```

Create it once with `New`/`giode.Pagination` and lay it out every frame.

## API

| Member | Description |
| --- | --- |
| `Pagination(total int, st ...Styles) *Pagination` | Creates a control over `total` pages, starting at page 1. |
| `Current() int` | The current page (1-indexed). |
| `SetCurrent(page int)` | Changes the current page without invoking OnChange, clamped to `[1, total]`. |
| `SetTotal(total int)` | Changes the page count, clamping the current page if needed. |
| `OnChange(fn func(page int)) *Pagination` | Handler for clicking a page button or prev/next. |

Styles apply to every button (prev/next and page numbers alike); the current page uses
`VariantContained`, every other button `VariantOutlined` - see [Button](button.md).

## Behavior

- With 7 or fewer total pages, every page number shows.
- With more, the row shows page 1, an ellipsis if there's a gap, up to 3 pages around the
  current one, another ellipsis if there's a gap, and the last page.
- Clicking prev/next past the first/last page is a no-op (no visual "disabled" state).
