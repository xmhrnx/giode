# Spinner

Stateless animated loading indicator.

```go
giode.Spinner(giode.Styles{Width: 24, Height: 24, Color: "#3b82f6"})
```

## Styles

| Property | Role |
| --- | --- |
| `Color` | Arc color. |
| `Width` / `Height` | Size; defaults to 32px. |

The spinner invalidates its own frames while visible; no state is required for the
animation itself.

## Hover

Chain `OnHover` to restyle the arc while the pointer is over it. This does need
persistent state, so create it once and reuse it - see [Hover](../elements.md#hover).

```go
loader := giode.Spinner(giode.Styles{Color: "#3b82f6"}).OnHover(giode.Styles{
	Color: "#60a5fa",
})
```
