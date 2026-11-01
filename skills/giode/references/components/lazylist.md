# LazyList

A vertical list of rows of any height that builds and lays out only the rows in or near the
viewport — giode's equivalent of React Native's `FlatList`, Flutter's `ListView.builder` or
Compose's `LazyColumn`. Use it for feeds, chats and anything whose rows differ in height; for
equal-height rows with keyboard navigation, [VirtualList](virtuallist.md) is simpler.

```go
list := giode.LazyList(
	func() int { return len(model.msgs) },              // count, read every frame
	func(i int) any { return model.msgs[i].ID },         // stable key per row
	func(i int) giode.Element { return messageRow(model.msgs[i]) },
	giode.Styles{Gap: 8},
).
	Version(func(i int) any { return model.msgs[i].Revision }).
	Anchor(giode.LazyAnchorBottom).
	OnReachTop(loadOlder)
```

The list fills the space it is given (give it a height, or `FlexGrow`). Each row gets the
list's width and its own natural height.

## How it keeps its place

The list remembers the key of the first row in view and where that row's top is, not a pixel
offset. So:

- Rows inserted **above** (older messages loaded by `OnReachTop`) don't move what's on screen.
- A row above the viewport that **changes height** doesn't move what's on screen.
- With `LazyAnchorBottom` the list starts at the end and stays there as rows are added or
  the last row grows (a streaming reply), until the user scrolls up. Once they scroll back to
  the end it follows again.

Heights of rows outside the viewport come from the last time they were laid out. Rows that
were never laid out use the average measured height, which is close enough for the
scrollbar. A long jump (dragging the scrollbar) skips over rows using those heights and
builds only the rows where it lands.

## Versions

`Version(fn)` gives each row a version: a revision number, a content hash, or the value itself
if it's comparable. A visible row whose version hasn't changed replays its drawing from cache
(like `Memo`), so scrolling and input don't rebuild it. An offscreen row keeps its measured
height until its version changes. Without `Version`, visible rows are rebuilt every frame, and
`Invalidate(key)` forgets a row's measured height when you know it changed.

## Methods

| Method | Effect |
| --- | --- |
| `Version(fn)` | Per-row version; see above. |
| `Anchor(a)` | `LazyAnchorTop` (default) or `LazyAnchorBottom`. |
| `OnReachTop(fn)`, `OnReachEnd(fn)` | Run when the first or last row comes into view, to load more. |
| `ScrollToKey(key, align)` | `LazyAlignStart`, `LazyAlignCenter` or `LazyAlignEnd`; applied on the next frame. Stops following the end. |
| `ScrollToBottom()` | Jumps to the end and, with `LazyAnchorBottom`, follows it again. |
| `Following()` | Whether the list is keeping to its end. |
| `VisibleRange()` | First and last row at least partly visible in the last frame. |
| `Invalidate(key)` | Forget a row's measured height. |
| `ScrollBy(dx, dy)`, `ScrollOffset()` | Scroll by pixels and read the (estimated) offset; used by the MCP server's `ui_scroll`. |

## Per-row state

Rows are only built near the viewport, so stateful widgets inside them should be looked up by
the row's key with `Keyed`, as with VirtualList.
