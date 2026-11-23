# Dialog

Centered modal dialog with a dimmed scrim. The open state lives in a `*bool` you own.

```go
open := new(bool)
dialog := giode.Dialog(open, func() giode.Element {
	return giode.Stack(giode.Styles{},
		giode.Text("Are you sure?"),
	)
}, giode.Styles{})

*open = true  // or false

// in the view, anywhere: it takes no space and paints over the window
giode.Stack(giode.Styles{}, page, dialog)
```

## API

| Member | Description |
| --- | --- |
| `Dialog(open *bool, content func() Element, st ...Styles) *Dialog` | Creates the dialog; styles apply to the panel. |
| `Opened() bool` | Whether the dialog is open. |
| `Dismissible(on bool) *Dialog` | Whether the scrim, Back and Escape close it (on by default). |
| `Top(dp int) *Dialog` | Places the panel dp below the window's top, centered horizontally, instead of in the middle: for search and command palettes that shouldn't move as their results grow. |
| `Padding(p Inset) *Dialog` | The panel's padding, zero included (`Padding(giode.Inset{})` lets content reach the edges; a zero `Padding` in the styles means the default). |

The dialog takes no space where it's placed, and a box's `Gap` leaves it out: while it's
open it paints over the whole window as an overlay, centered in the window rather than
in its parent.

Clicking the scrim, pressing Escape or pressing the system Back button (Android) sets
`*open` to false. Turn this off with `Dismissible(false)` for a dialog the user must
answer through its own buttons. Back and Escape go to the topmost open overlay only. Panel styles follow `Box`: `Background`,
`BorderRadius` (default 12), `Padding` (default 20), `BoxShadow`.
