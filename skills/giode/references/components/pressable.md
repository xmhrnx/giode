# Pressable

Makes any content clickable, not just a text button. The content is rebuilt every frame
with the current `PressState`, so it can restyle itself.

```go
card := giode.Pressable(func(s giode.PressState) giode.Element {
	bg := "theme.surface"
	switch {
	case s.Pressed:
		bg = "theme.border"
	case s.Hovered:
		bg = "theme.surface/0.8"
	}
	return giode.Box(giode.Styles{Background: bg, Padding: giode.UniformInset(12), BorderRadius: 8},
		giode.Text("Open project", giode.Styles{Color: "theme.text"}))
}).
	Label("Open project").
	OnClick(func(e giode.ClickEvent) {
		if e.Ctrl() {
			openInNewWindow()
			return
		}
		open()
	}).
	OnDoubleClick(func(giode.ClickEvent) { rename() }).
	OnSecondaryClick(menu.OpenAtClick)
```

`PressState` has `Hovered`, `Pressed`, `Focused`, `Disabled` and `Dragging`.

## ClickEvent

| Field | Meaning |
| --- | --- |
| `Clicks` | 1 for a single click, 2 for a double click, ... |
| `Button` | `MousePrimary`, `MouseSecondary`, `MouseMiddle` |
| `Modifiers` | keys held; `e.Ctrl()` (Command on macOS) and `e.Shift()` |
| `Source` | `ClickPointer`, `ClickKeyboard`, `ClickProgrammatic` |
| `Position` | where the pointer was, relative to the pressable, in pixels |
| `PositionDp` | the same in dp, the unit of `Styles` |
| `WindowPosition` | where the pointer was in the window, in pixels; for keyboard and programmatic clicks, the pressable's bottom-left corner. `Menu.OpenAtClick(e)` opens a menu there. |

`OnLongPress(fn)` handles a touch held still on the element for half a second (the touch
screen's way to open context actions; the release that follows isn't a click). With a
mouse, use `OnSecondaryClick`. `Styles(st)` replaces the box styles from the next frame.

`OnClick` sees every click; `OnDoubleClick` and `OnSecondaryClick` run in addition for
those kinds.

## Dragging

```go
tab := giode.Pressable(tabContent).
	OnClick(func(giode.ClickEvent) { activate(id) }).
	OnSecondaryClick(tabMenu.OpenAtClick).
	OnDragStart(func(e giode.DragEvent) { dragging = id }).
	OnDragMove(func(e giode.DragEvent) { offset = e.DeltaDp.X; moveTab(id, e.WindowPosition) }).
	OnDragEnd(func(e giode.DragEvent) {
		dragging = nil
		if !e.Cancelled {
			drop(id)
		}
	})
```

A press that moves more than 6dp, with any button or a finger, is a drag: `OnDragStart`
runs once (with the press position), `OnDragMove` on every move, and `OnDragEnd` on the
release. **A press that became a drag doesn't click**, so the same element still clicks
when it's only tapped; `Dragging()` and `PressState.Dragging` report a drag in progress.
The drag goes on when the pointer leaves the element, and once it has started, an
enclosing `Scroll` or a frameless window's drag area doesn't take it. A finger moving
along a `Scroll`'s axis scrolls it before it moves far enough to start a drag.

| `DragEvent` field | Meaning |
| --- | --- |
| `Position` / `PositionDp` | where the pointer is, relative to the pressable, in pixels / dp |
| `WindowPosition` | where the pointer is in the window, in pixels |
| `DeltaDp` | how far the pointer has moved since the press, in dp, measured in the window - right even while the element moves with the drag |
| `Button`, `Modifiers` | the button held (`MousePrimary` for touch) and the keys held |
| `Touch` | set for a finger |
| `Cancelled` | on `OnDragEnd`: the drag ended without a release (the system took the pointer, or the pressable was disabled) - treat it as no drop |

Setting a drag handler on a box made clickable with `OnClick` (through `box.Pressable()`)
turns off selecting its text by dragging.

## Behavior

- The hit area sits **beneath** the content, like Gio's `widget.Clickable`, so clickable
  children (an expand arrow, a delete button) keep their own clicks.
- Pointer cursor on hover (override with the `Cursor` style).
- On a touch screen, `PressState.Pressed` (and `Hovered`) is set while the finger is down,
  so content styled for either gives feedback. Lifting the finger after sliding it off
  the element is not a click, as with the mouse.
- Tab focuses it; Enter or Space activates it. Disable with `Focusable(false)`.
- `Disabled(true)`: no events, no focus, default cursor; the content sees
  `PressState.Disabled`.
- It stays clickable inside a frameless window's drag area.
- Automation (the MCP server) sees it as a button and can click it; set `Label` so it has
  a name.
- `Click()` runs the handlers programmatically (on the UI goroutine).

The optional styles argument wraps the content in a box: padding, background, `FlexGrow`,
`Width`, ...

## Box.OnClick

For simple cases, any box can be made clickable:

```go
row := giode.Box(giode.Styles{Padding: giode.UniformInset(8), Cursor: giode.CursorPointer}, label).
	OnClick(func(giode.ClickEvent) { selectRow(id) })
```

Like `OnHover`, press state lives in the element, so the box must be created once and
reused, or looked up by key with `Keyed` inside lists. `box.Pressable()` returns the
underlying Pressable for labels, double-click handlers and disabling.
