# Progress

Stateless determinate bar. Build it every frame.

```go
giode.Progress(0.35, giode.Styles{Color: "#3b82f6"})
```

| Argument | Description |
| --- | --- |
| `value` | Progress in [0, 1]. |
| `st` | Optional styles. |

## Styles

| Property | Role |
| --- | --- |
| `Color` | Fill. |
| `Background` | Track. |
| `Height` | Bar height; defaults to 8px. |

The bar expands to the available width.

## Hover

Chain `OnHover` to restyle the bar while the pointer is over it (create it once and
reuse it, like any element with hover styles):

```go
bar := giode.Progress(0.35, giode.Styles{Color: "#3b82f6"}).OnHover(giode.Styles{
	Color: "#60a5fa",
})
```

See [Hover](../elements.md#hover) for the full mechanism.
