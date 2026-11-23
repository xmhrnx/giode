# SplitPane

`giode.SplitPane(first, second)` shows two panes with a divider between them that the
user drags to share the space, such as a sidebar and an editor, or an editor and a
terminal. Create it once and lay it out every frame. It fills the space it's given.

```go
split := giode.SplitPane(
	func() giode.Element { return fileTree },
	func() giode.Element { return editor },
).FirstSize(240).OnResize(func(ratio, firstDp float32) { prefs.SidebarWidth = firstDp })

work := giode.SplitPane(
	func() giode.Element { return editor },
	func() giode.Element { return terminal },
).Axis(giode.SplitVertical).Ratio(0.7)
```

| Method | |
|---|---|
| `Axis(giode.SplitHorizontal \| giode.SplitVertical)` | Side by side (the default; the first pane goes at the start, so on the right in a right-to-left locale), or one above the other. |
| `Ratio(r)` | The first pane's share of the space, 0 to 1; 0.5 by default. It keeps that share as the space changes. |
| `FirstSize(dp)` | The first pane keeps this size as the space changes, like a sidebar; the second pane takes the rest. |
| `MinSizes(first, second)` | How small each pane may get, in dp; 48 by default. The second pane's minimum wins when both can't fit. |
| `DividerColor(c)` | The divider line's color; `theme.border` by default, `theme.primary` while hovered, dragged or focused. |
| `OnResize(fn)` | Runs as the user moves the divider, with the first pane's share and its size in dp, to remember them. |
| `Content(first, second)` | Replaces the panes. |
| `Share()`, `Dragging()` | The first pane's share in the last frame, and whether the divider is being dragged. |

Dragging the divider resizes the panes, with a resize cursor over it. A double-click puts
back the size it started with. Tab focuses the divider: the arrow keys move it 16dp
(64dp with Shift), and Home and End move it to either end. Screen readers see it as a
slider whose value is the first pane's share.
