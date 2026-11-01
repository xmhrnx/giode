# Scroll

Vertical scroll container. The content closure is rebuilt every frame and may be taller
than the viewport. The mouse wheel, touch drags and the scrollbar thumb all scroll.

```go
scroller := giode.Scroll(func() giode.Element {
	// build the content fresh every frame
	return giode.Stack(giode.Styles{Gap: 8}, items...)
}, giode.Styles{ScrollBar: giode.ScrollBar{ThumbColor: "#3b82f6"}})

// in the view:
giode.Box(giode.Styles{Height: 300}, scroller)
```

## API

| Member | Description |
| --- | --- |
| `Scroll(content func() Element, st ...Styles) *Scroll` | Creates the container. |
| `ScrollTo(v float32)` | Sets the vertical offset in px. |
| `Offset() float32` | Current offset. |
| `ContentHeight() int` | Total content height, as measured on the last frame. |
| `ViewportHeight() int` | Visible viewport height, as measured on the last frame. |
| `Progress() float32` | How far scrolled, from 0 (top) to 1 (bottom); 0 when there's nothing to scroll. |
| `AtBottom() bool` | Whether the scroll is at (or past) its maximum offset - also true when the content already fits the viewport. |
| `Visible(top, height int) bool` | Whether the vertical range `[top, top+height)` - in the content's own coordinate space, e.g. a running offset tracked while building a long list - is at least partially within the current viewport. |
| `OnReachBottom(fn func()) *Scroll` | Handler fired once when the scroll reaches its maximum offset (edge-triggered - fires again only after scrolling away and back). |
| `Anchor(a ScrollAnchor) *Scroll` | `AnchorBottom` keeps a scroll that's at the bottom there as content grows (a chat following a streaming reply), stops once the user scrolls up, and resumes when they scroll back down. It starts at the bottom. |
| `ScrollToBottom()` | Scrolls to the end, and resumes following with `AnchorBottom`. |
| `ScrollBy(dx, dy float32)` / `ScrollOffset() (x, y float32)` | Scroll by pixels as a wheel would (positive `dy` scrolls down), and read both offsets. The MCP server's `ui_scroll` uses them. |
| `FitContent(on bool) *Scroll` | The viewport hugs the content up to the available space instead of filling it - a sheet with three rows stays three rows tall. |
| `Axis(a ScrollAxis) *Scroll` | `Vertical` (default), `Horizontal`, or `Vertical\|Horizontal`. A horizontal-only scroll is as tall as its content, like a row - for wide code and tables. |
| `OffsetX() float32` / `ScrollToX(v float32)` | The horizontal offset. |
| `OnScroll(fn func(offset, max float32)) *Scroll` | Called after the offset changes, with the offset and its maximum in px (horizontal ones for a horizontal-only scroll). |
| `Styles(st Styles) *Scroll` | Replaces the styles from the next frame. |

```go
chat := giode.Scroll(messages).Anchor(giode.AnchorBottom)
code := giode.Scroll(codeBlock).Axis(giode.Horizontal)
```

Scrolls nest by axis: a horizontal code block inside a vertical chat scrolls sideways
with a sideways drag and passes vertical drags and the mouse wheel on to the chat.

## Detecting scroll position

An intersection-observer-like pair for infinite scroll and lazy rendering:

```go
list := giode.Scroll(func() giode.Element {
	items := make([]giode.Element, 0, loaded)
	for i, top := 0, 0; i < loaded; i, top = i+1, top+itemHeight {
		if !list.Visible(top, itemHeight) {
			items = append(items, placeholder) // skip expensive content off-screen
			continue
		}
		items = append(items, renderItem(i))
	}
	return giode.Stack(giode.Styles{}, items...)
}, giode.Styles{})

list.OnReachBottom(func() {
	loaded += pageSize // fetch/reveal the next page
})
```

`Visible` and `AtBottom` reflect the scroll offset as of the start of this Layout call,
one frame behind the content closure's own not-yet-measured output - fine for both
patterns above: a placeholder swaps to real content one frame after scrolling it into
view, and "reached bottom" is a one-off trigger, not a per-frame read.

## Styles

| Property | Role |
| --- | --- |
| `Background` | Container fill. |
| `ScrollBar` | Scrollbar customization: `Width`, `TrackColor`, `ThumbColor`, `Radius`, `MinThumbLength`, `Inset`, `AutoHide`, and `Hidden`, `HiddenMaxWidth`, `HiddenOn` (below). |

Interactive children inside the content keep receiving events. For very long lists use
[VirtualList](virtuallist.md) or [LazyList](lazylist.md).

## A quieter scrollbar

```go
giode.Styles{ScrollBar: giode.ScrollBar{
	Width:      6,
	ThumbColor: "#ffffff88",
	TrackColor: "transparent", // no track: the thumb alone
	Inset:      4,             // 4dp in from the right edge, and from the top and bottom
	AutoHide:   true,          // shown while scrolling, hovered or dragged; fades out after
}}
```

An unset `TrackColor` draws the default track; any color with zero alpha draws none.
With `AutoHide` the bar appears when the content scrolls or the pointer is over the bar,
stays a second after, and fades out over 300ms, as macOS, Windows 11 and browsers do.

## Hiding the scrollbar

A hidden scrollbar isn't drawn; the content still scrolls with the wheel, touch, the
keyboard and `ScrollBy`. The same fields work on `Scroll`, `VirtualList`, `LazyList` and
`TreeView`.

```go
// Never show it.
giode.Styles{ScrollBar: giode.ScrollBar{Hidden: true}}

// Hide it while the window is 600dp wide or less (phones), show it on wider windows.
giode.Styles{ScrollBar: giode.ScrollBar{HiddenMaxWidth: 600}}

// Hide it while the enclosing Responsive matched the "compact" breakpoint
// (several names can be given, separated by commas: "compact,medium").
giode.Styles{ScrollBar: giode.ScrollBar{HiddenOn: "compact"}}
```

`HiddenMaxWidth` is compared with the window's width, so a list shared between the desktop
and mobile layouts behaves the same wherever it sits in the layout. `HiddenOn` follows your
own `Responsive` breakpoints, so the scrollbar switches exactly when the layout does. Since
the view is rebuilt every frame, any condition also works: `Hidden: giode.Compact()`.
