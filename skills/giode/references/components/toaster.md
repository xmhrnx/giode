# Toaster

Small notifications that pop up in a corner of the window and disappear on their own.

```go
toasts := giode.Toaster().Placement(giode.ToastBottomEnd).Duration(4 * time.Second).Max(3)

app.Run(func() giode.Element {
	return giode.Stack(giode.Styles{Width: -1, Height: -1}, content(), toasts)
})

// anywhere, from any goroutine:
toasts.Show(giode.ToastSuccess, "Saved")
id := toasts.ShowAction(giode.ToastWarning, "File deleted", "Undo", restore)
toasts.Dismiss(id)
```

Lay the toaster out once per frame anywhere in the view: it takes no space and paints above
everything else, and only the toasts themselves take input.

| Setting | Default | |
| --- | --- | --- |
| `Placement(p)` | `ToastBottomEnd` | The bottom right, or bottom left in a right-to-left locale. Also `ToastBottomStart`, `ToastTopEnd`, `ToastTopStart`, which follow the reading direction too, and the fixed `ToastBottomRight`, `ToastBottomLeft`, `ToastBottomCenter`, `ToastTopRight`, `ToastTopLeft`, `ToastTopCenter`. |
| `Duration(d)` | 5s | Zero or less keeps toasts until dismissed. |
| `Max(n)` | 3 | How many show at once; the rest wait their turn. |
| `Inset(properties.Inset)` | none | Extra dp from each window edge, to keep toasts clear of a custom title bar or a control bar. |

Kinds — `ToastInfo`, `ToastSuccess`, `ToastWarning`, `ToastError` — color the accent stripe
and action from the theme tokens (`theme.info`, `theme.success`, `theme.warning`,
`theme.danger`); the toast itself uses `theme.surface`, `theme.text` and `theme.border`
unless the styles argument overrides `Background`, `Color`, `BorderColor`, `BorderRadius`,
`Width` or `FontSize`.

- The timer pauses while the pointer is over a toast.
- Expiry uses timed frame requests (`kit.RequestFrameAt`), not goroutines or timers.
- Toasts slide and fade in, and collapse out so the others move smoothly.
- `Show`, `ShowAction`, `Dismiss` and `DismissAll` are safe from any goroutine; the action
  callback runs on the UI goroutine.
- Automation (the MCP server) sees each message, its action and its dismiss button.
