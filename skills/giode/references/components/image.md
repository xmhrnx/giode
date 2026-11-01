# Image

Stateless image with fit modes.

```go
giode.Image(img, giode.Styles{
	Width: 120, Height: 120,
	Fit: giode.FitCover,
	BorderRadius: 12,
})
```

| Argument | Description |
| --- | --- |
| `img` | Any `image.Image` (decode PNG/JPEG with the standard library). |
| `st` | Optional styles. |

## Styles

| Property | Role |
| --- | --- |
| `Width` / `Height` | Box size; natural size when unset. |
| `Fit` | `FitContain` (default), `FitCover`, `FitStretch`, `FitNone`. |
| `BorderRadius` | Rounded corners. |
| `Opacity` | Applied on top of the image; useful with `OnHover` to dim/brighten it. |

## Hover

Chain `OnHover` to change `Opacity` or `BorderRadius` while the pointer is over the
image. Since the hover state lives in the element, create it once and reuse it:

```go
thumb := giode.Image(img, giode.Styles{Width: 120, Height: 120}).OnHover(giode.Styles{
	Opacity: 0.8,
	Cursor:  giode.CursorPointer,
})
```

See [Hover](../elements.md#hover) for the full mechanism.
