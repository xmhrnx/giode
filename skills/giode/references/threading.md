# Threading

Everything that touches the UI runs on **one goroutine, the UI goroutine**:

- the view function passed to `App.Run`;
- every `Layout` method;
- content closures (`Scroll`, `VirtualList`, `Tooltip`, ...);
- event handlers: `OnClick`, `OnChange`, `Submit`, menu actions, ...

Data that is only changed there needs no locks. The rule is: **change the model on the UI
goroutine, and read it anywhere in the view.**

## From other goroutines: Post

Background work — a network request, a file read, a timer — hands its result to the UI
goroutine with `app.Post`:

```go
go func() {
	items, err := fetchItems()
	app.Post(func() {
		model.items, model.err = items, err // runs on the UI goroutine
	})
}()
```

`Post` doesn't wait: it queues the function, requests a frame, and the function runs before
that frame is laid out. Calls run in the order posted. `app.RunOnUI(ctx, fn)` does the same
but waits until `fn` has run.

`State` values (`giode.NewState`) are safe from any goroutine on their own, and redraw when
set.

## During a frame: AfterLayout

A handler runs in the middle of laying out the frame. If it changes data the rest of the
frame is still reading — deleting the list item whose button was clicked, say — defer the
change:

```go
remove.OnClick(func() {
	giode.AfterLayout(func() { model.delete(id) })
})
```

`AfterLayout` functions run once the frame's layout is done, still on the UI goroutine, and
another frame follows so the change shows up immediately.

## Sizes outside layout

Styles use dp. Code that has no layout context can convert with the last frame's display
scale: `giode.DisplayScale()`, `giode.DpToPx(v)`, `giode.PxToDp(px)`.
