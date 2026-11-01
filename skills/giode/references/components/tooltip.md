# Tooltip

Wraps any element as a trigger: hovering it with a mouse, or long-pressing it on a touch
screen (half a second), shows a small panel just below, rendered
above everything else regardless of nesting (see [extending.md](../extending.md#rendering-above-everything-portals)).

```go
trigger := giode.Button("Hover me", giode.Styles{Background: "gray"})
tip := giode.Tooltip(trigger, "A helpful hint")

// in the view, lay out tip in trigger's place:
giode.Stack(giode.Styles{}, tip)
```

Create it once with `New`/`giode.Tooltip` - it wraps a stateful trigger, so it must be
laid out every frame like any other component, not rebuilt inside the view function.

## API

| Member | Description |
| --- | --- |
| `Tooltip(trigger Element, content string, st ...Styles) *Tooltip` | Wraps trigger. |
| `SetContent(content string)` | Replaces the tooltip text. |
| `Placement(p Placement) *Tooltip` | `PlaceAuto` (default: below the trigger, above it when it doesn't fit, never over it), `PlaceBelow` or `PlaceAbove`. |
| `Hovered() bool` | Whether a mouse was over the trigger on the last frame. |
| `Visible() bool` | Whether the tooltip was showing on the last frame, from hover or a long-press. |

After a long-press the tooltip stays for 1.5 seconds once the finger lifts. A trigger that
is itself clickable still receives the click when the finger lifts.

## Styles

| Property | Role |
| --- | --- |
| `Background` | Panel fill; defaults to a dark slate. |
| `Color` | Text color; defaults to near-white. |
| `FontSize` | Text size; defaults to 13. |
