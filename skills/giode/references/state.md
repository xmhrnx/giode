# Global state

`giode.State[T]` is a minimal, generic observable value: read and write it from
anywhere, including a goroutine outside the render loop (a network response, a timer),
not just from within a `Layout` call.

```go
counter := giode.NewState(0)

btn := giode.Button("Increment").OnClick(func() {
	counter.Update(func(v int) int { return v + 1 })
})

// elsewhere - a goroutine, a network callback, a timer:
go func() {
	result := fetchSomething()
	counter.Set(result)
}()

// in the view:
giode.Text(fmt.Sprintf("Count: %d", counter.Get()))
```

Unlike every other piece of state in giode (a `Button`'s click count, a `Scroll`'s
offset), which lives inside one component and only ever changes from within a `Layout`
call, `State` is meant to be shared across the app and written from anywhere - `Set`/
`Update` request a redraw themselves (via the app window's `Invalidate`, wired up
automatically by `App.New`), so a background goroutine's update shows up without
needing its own `gtx.Execute(op.InvalidateCmd{})`.

## API

| Member | Description |
| --- | --- |
| `NewState[T](initial T) *State[T]` | Creates a State holding `initial`. |
| `Get() T` | Current value. |
| `Set(v T)` | Replaces the value, notifies watchers, requests a redraw. |
| `Update(fn func(current T) T)` | Atomically replaces the value with `fn(current)` - safe for a read-modify-write under concurrent access (e.g. an increment) where a separate `Get` then `Set` could race. |
| `Watch(fn func(T)) (unsubscribe func())` | Registers a callback for every future `Set`/`Update`. |

`State` is safe for concurrent use - `Get`, `Set`, `Update` and `Watch` can all be called
from multiple goroutines at once.

## When to reach for it

Most state still belongs inside one stateful component (a counter local to one page,
say) - a `State` value's job is specifically being shared across the view, or written
from outside the render loop:

- A value several unrelated parts of the view need to read (a signed-in user, a color
  theme flag, a shopping cart count in a header badge).
- A value updated by a background goroutine - the case its own redraw-triggering exists
  for.

For UI-only interactions confined to one component (a click count next to one button, a
dropdown's own selection), the component's own field is simpler - see
[Extending the library](extending.md).

## Per-item state for dynamic lists

Stateful components must outlive a frame, which is easy for a fixed UI — create them once —
but not for a list whose items come and go. `Keyed` gives each item its own state, looked up
by a stable key:

```go
func fileRows(files []File) []giode.Element {
	rows := make([]giode.Element, 0, len(files))
	for _, f := range files {
		f := f
		row := giode.Keyed("files", f.ID, func() *giode.PressableElement {
			return giode.Pressable(func(s giode.PressState) giode.Element {
				return fileRow(f, s)
			}).OnClick(func(giode.ClickEvent) { open(f) })
		})
		rows = append(rows, row)
	}
	return rows
}
```

- Use IDs, not indexes: with indexes, removing an item hands its state to the next one.
- The scope (`"files"`) keeps unrelated lists apart when their keys could collide.
- An entry nobody asked for in 60 frames is dropped (`SetKeyedRetention` changes that; frames
  only happen when something changes, so it isn't a fixed time). `ForgetKeyed` drops one
  right away.

Inside a stateful component, a typed `KeyedCache` does the same without global scopes:

```go
type fileList struct {
	rows *giode.KeyedCache[string, *giode.PressableElement]
}

l.rows = giode.NewKeyedCache[string, *giode.PressableElement]().Retain(120).
	OnEvict(func(id string, p *giode.PressableElement) { /* release resources */ })

row := l.rows.Get(f.ID, func() *giode.PressableElement { ... })
```
