# Zoomable

Pinch-to-zoom and pan for a page viewer, a photo or a crop editor.

```go
viewer := giode.Zoomable(func() giode.Element {
	return giode.Image(page, giode.Styles{Width: -1})
}).Limits(1, 6)
```

Create it once (or with `giode.Keyed`) and lay it out every frame. It fills the space it's
given and centres the content, laid out to fit, at scale 1.

| Gesture | Effect |
| --- | --- |
| Two fingers | Pinch to zoom around their midpoint; moving them moves the content. |
| One finger | Pans once the content is zoomed in past the view. |
| Double tap | Zooms in at the tap to `DoubleTapScale` (2.5), or back out to 1. |
| Ctrl / Cmd + wheel or trackpad scroll | Zooms at the pointer. |
| Wheel | Pans a zoomed view, then lets a list around it scroll. With `WheelZoom(true)` it zooms instead. |

| Member | Description |
| --- | --- |
| `Limits(min, max)` | Smallest and largest scale (default 1 to 5). |
| `DoubleTapScale(s)` | Scale a double tap zooms to; 0 turns double taps off. |
| `WheelZoom(on)` | The wheel zooms without a modifier, as image viewers do. |
| `OnChange(fn(scale, offset))` | Called when the scale or offset changes. |
| `Scale()`, `Offset()`, `Zoomed()` | Current state. |
| `SetScale(s)`, `ZoomIn()`, `ZoomOut()`, `Reset()` | Animated zoom around the centre, for buttons and shortcuts. |
| `ToContent(p)`, `ToView(p)` | Convert points between the view and the content's own coordinates. |

The content's own gestures stay with it: a crop handle's drag or a button inside it gets its
input first, transformed to the content's coordinates at any scale, and Zoomable sees only the
touches the content doesn't take. Portals opened from inside zoomed content (a menu or a
tooltip) are placed as if it weren't zoomed.
