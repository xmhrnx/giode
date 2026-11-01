# Drawer

Slide-out side panel with a dimmed scrim, animated over 200ms. The open state lives in a
`*bool` you own, so opening and closing works from anywhere.

```go
open := new(bool)
drawer := giode.Drawer(open, giode.SideLeft, 260, func() giode.Element {
	return giode.Stack(giode.Styles{Padding: giode.UniformInset(16)},
		giode.Text("Drawer content"),
	)
}, giode.Styles{Background: "#1e293b"})

*open = true  // or false

// in the view, anywhere: it takes no space and paints over the window
giode.Stack(giode.Styles{}, page, drawer)
```

The drawer takes no space in the layout. While it's open (or sliding), it paints over the
whole window as an overlay, so it can sit anywhere in the tree.

## API

| Member | Description |
| --- | --- |
| `Drawer(open *bool, side Side, width int, content func() Element, st ...Styles) *Drawer` | Creates the drawer; `side` is `SideLeft` or `SideRight`, or `SideStart`/`SideEnd`, which follow the reading direction (`SideStart` is the right in a right-to-left locale). |
| `Opened() bool` | Whether the drawer is open. |
| `Dismissible(on bool) *Drawer` | Whether the scrim, Back, Escape and dragging close it (on by default). |
| `SwipeToOpen(on bool) *Drawer` | Drag from the window edge to open it (off by default). |
| `Styles(st Styles) *Drawer` | Replaces the panel styles from the next frame. |

Clicking the scrim, pressing Escape, pressing the system Back button (Android) or dragging
the panel toward its edge sets `*open` to false, unless `Dismissible(false)` was set.
With `SwipeToOpen(true)`, dragging from the drawer's edge of the window opens it; on
Android with gesture navigation that edge swipe is also the system Back gesture, which is
why it's off by default.
