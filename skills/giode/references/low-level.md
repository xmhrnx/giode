# Low-level access

Raw Gio stays reachable at every layer.

## Raw elements

Drop to plain Gio at any point in the tree:

```go
giode.Box(giode.Styles{},
	giode.Raw(func(gtx layout.Context) layout.Dimensions {
		// any Gio layout, widget, paint or input code
		return layout.Dimensions{}
	}),
)
```

## Raw run loop

Skip the element model entirely:

```go
app := giode.New(giode.Config{...})
app.RunRaw(func(gtx layout.Context) layout.Dimensions {
	// a complete Gio frame
	return layout.Dimensions{}
})
```

## The window

`app.Window()` returns the underlying `*app.Window` for options, actions and events not
covered by `Config` and the action methods.

## GLFW and embedded windows

Only the root `App` layer depends on `gioui.org/app`. Every element, component, style
property, the router, icons and fonts are window-agnostic: they only need a
`layout.Context`. Rendering through a custom window (GLFW, Ebitengine, an embedded
surface) therefore works unchanged.

Import `github.com/mehran9675/giode/ui` instead of the root package: it has the same
names (`ui.Box`, `ui.Button`, `ui.Styles`, …; the root package's are generated from it)
without `App`, so Gio's window layer isn't linked. `github.com/mehran9675/giode/host` has
the window-free services: `KeepScreenOn`, `KeepAwake`, `LockInstance` (the
single-instance lock) and `ServeClipboard`.

Build the context yourself each frame and lay it out with `ui.Frame`, which runs the
steps `App` runs (frame start and end, window size and metric, portals, the Back key,
after-layout callbacks); skipping any of them makes menus, tooltips, keyed state or
toasts misbehave:

```go
import (
	"github.com/mehran9675/giode/host"
	"github.com/mehran9675/giode/ui"
)

ui.SetInvalidator(glfw.PostEmptyEvent) // State changes and toasts from other goroutines
lock := host.LockInstance("com.example.player")
if lock.Running() {
	lock.Forward(os.Args[1:])
	return
}
defer lock.Release()
lock.Serve(func(args []string) { openFiles(args) })

// per frame:
ops.Reset()
gtx := layout.Context{
	Ops:         &ops,
	Now:         time.Now(),
	Source:      router.Source(),            // input.Router fed by your window callbacks
	Metric:      unit.Metric{PxPerDp: scale, PxPerSp: scale},
	Constraints: layout.Exact(image.Pt(fbw, fbh)), // framebuffer size
}
paint.Fill(gtx.Ops, background)
again := ui.Frame(gtx, view())
_ = gpuCtx.Frame(gtx.Ops, gpu.OpenGLRenderTarget{}, image.Pt(fbw, fbh))
router.Frame(gtx.Ops)                       // hit-testing for the next frame
host.ServeClipboard(&router, window.GetClipboardString, window.SetClipboardString)
window.SwapBuffers()
if again {
	glfw.PostEmptyEvent()
}
```

Notes for custom loops:

- Queue pointer/key events into `input.Router` in the same pixel space as the layout
  constraints; priority, pass-through and grab semantics are all handled by the router.
- The context menu works manually: call `menu.Context(gtx)` **before** the view and
  `menu.Layout(gtx)` **after** it, inside the element given to `ui.Frame` (the
  equivalent of `SetContextMenu`).
- `Config.Draggable` relies on Gio's window driver (`system.ActionInputOp(ActionMove)`)
  and is a no-op under GLFW — use native window dragging there instead.
- The `Config` window options and the `App` action methods (fullscreen, minimize,
  center, always-on-top, …) are `app.Window`-only.
- Animations (Spinner, Drawer, Dialog) request frames with `op.InvalidateCmd`. A
  continuous render loop ignores that naturally; an event-driven loop must poll
  `router.WakeupTime()` or the animations freeze.
- To draw video or 3D under the UI without a custom host, see `Config.Underlay` in
  [Window](window.md).

## Direct package imports

The root package is a re-export convenience. Everything is also importable from its
natural package:

```go
import (
	"github.com/mehran9675/giode/elements"
	"github.com/mehran9675/giode/components/menu"
	"github.com/mehran9675/giode/styles/properties"
)
```
