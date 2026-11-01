# Accordion

Stateful list of expand/collapse sections.

```go
acc := giode.Accordion(giode.Styles{}).
	Item("Section one", func() giode.Element { return giode.Text("...") }).
	Item("Section two", func() giode.Element { return giode.Text("...") })

// in the view:
giode.Stack(giode.Styles{}, acc)
```

Create it once with `New`/`giode.Accordion` and lay it out every frame - sections may be
added at any time.

## API

| Member | Description |
| --- | --- |
| `Accordion(st ...Styles) *Accordion` | Creates an empty accordion. |
| `Item(label string, content func() Element) *Accordion` | Appends a section; content is built lazily, only while expanded. |
| `SingleOpen(v bool) *Accordion` | Opening one section closes any other open one (default: sections expand independently). |
| `Expanded(i int) bool` | Whether section `i` is open. |
| `SetExpanded(i int, v bool)` | Opens or closes section `i` programmatically. |

Styles: `Color` (label and chevron), `BorderColor` (divider between sections).

## Behavior

- Clicking a header toggles that section; the chevron points right when closed, down
  when open.
- Collapsing is instant, not animated.
- A section's `content` closure only runs while that section is expanded, so building it
  can be as expensive as the content actually needs.
