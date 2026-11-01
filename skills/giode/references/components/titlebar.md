# TitleBar

A custom window title bar for frameless windows: a drag handle with the title, an optional
icon and menus, and minimize / maximize-restore / close buttons.

```go
app := giode.New(giode.Config{
	Title:     "My app",
	Frameless: true, // drop the OS title bar
	Resizable: true,
})

bar := giode.TitleBar(app, "My app", giode.Styles{BorderWidth: 1, BorderColor: "theme.border"}).
	Icon(giode.Icon("logo", giode.Styles{Width: 16})).
	Leading(fileMenu.Trigger(fileButton)). // menus after the title
	Trailing(accountButton)                 // controls before the window buttons

app.Run(func() giode.Element {
	return giode.Stack(giode.Styles{Width: -1, Height: -1}, bar, content())
})
```

Create it once and lay it out every frame at the top of the window. Don't set
`Config.Draggable`: the bar is the drag handle, so only it moves the window.

## Behavior

- Dragging empty bar space moves the window. On Windows the bar is a real caption, so
  double-clicking maximizes and dragging to a screen edge snaps (Aero Snap); elsewhere the
  bar handles double-click itself.
- The maximize button shows a restore glyph while the window is maximized, following
  `App.WindowState()`, whether the change came from the app or the user.
- Everything placed in `Leading`/`Trailing` stays clickable inside the drag area.
- When the window loses focus, the title and buttons dim, as native title bars do.
- A frameless window keeps its resize borders on Windows.

## Options

| Method | Effect |
| --- | --- |
| `Icon(el)` | Element before the title. |
| `Leading(els...)` | Elements after the title: menus, tabs, search. |
| `Trailing(els...)` | Elements before the window buttons. |
| `Variant(v)` | `TitleBarWindows` (default: square buttons on the right, red close hover) or `TitleBarMac` (traffic lights on the left, centered title). |
| `Hide(buttons...)` | Removes `TitleBarMinimize`, `TitleBarMaximize` and/or `TitleBarClose`. |
| `OnClose(fn func() bool)` | Runs on close; return `false` to keep the window open (unsaved changes). |
| `Title(s)` | Replaces the title. |
| `Styles(st)` | Replaces the styles: solid in a window, a transparent fade over fullscreen video. |
| `Press(button)` | Performs a button's action programmatically. |

Styles: `Background` (default `theme.surface`), `Color` (title and glyphs, default
`theme.text`), `Height` (default 36dp, 28dp for the Mac look), `FontSize`, `FontWeight`,
`Padding`, and `BorderWidth` + `BorderColor` for a line under the bar.

## Window state

The bar uses these `App` methods, which are also available on their own:

```go
app.WindowState()   // WindowNormal, WindowMinimized, WindowMaximized, WindowFullscreen
app.Maximized()
app.ToggleMaximize()
app.Focused()

giode.Config{OnWindowStateChange: func(s giode.WindowState) { ... }}
```

To build your own bar, mark its area with `kit.DragArea(gtx, size)` *before* laying out its
contents; buttons inside carve themselves out with `kit.BlockDrag`.
