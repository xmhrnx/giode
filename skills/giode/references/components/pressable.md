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
	OnSecondaryClick(func(e giode.ClickEvent) { menu.OpenAt(...) })
```

`PressState` has `Hovered`, `Pressed`, `Focused` and `Disabled`.

## ClickEvent

| Field | Meaning |
| --- | --- |
| `Clicks` | 1 for a single click, 2 for a double click, ... |
| `Button` | `MousePrimary`, `MouseSecondary`, `MouseMiddle` |
| `Modifiers` | keys held; `e.Ctrl()` (Command on macOS) and `e.Shift()` |
| `Source` | `ClickPointer`, `ClickKeyboard`, `ClickProgrammatic` |
| `Position` | where the pointer was, relative to the pressable, in pixels |
| `PositionDp` | the same in dp, the unit of `Styles` |

`OnLongPress(fn)` handles a touch held still on the element for half a second (the touch
screen's way to open context actions; the release that follows isn't a click). With a
mouse, use `OnSecondaryClick`. `Styles(st)` replaces the box styles from the next frame.

`OnClick` sees every click; `OnDoubleClick` and `OnSecondaryClick` run in addition for
those kinds.

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
