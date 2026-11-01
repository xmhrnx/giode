# Extending the library

Everything in Giode is an `Element`:

```go
type Element interface {
	Layout(gtx layout.Context) layout.Dimensions
}
```

Any type with a `Layout` method is a first-class Giode component — it can be returned
from a view, placed inside a `Box`/`Row`/`Stack` and re-exported next to the built-ins. On top of
that interface, Giode offers a small kit of helpers that the built-in components
themselves are made of.

## Quick starts

| Need | Use |
| --- | --- |
| One-off raw Gio code | `giode.Raw(func(gtx layout.Context) layout.Dimensions { ... })` |
| Stateless visual | A struct with a `Layout` method built from `Box`/`Row`/`Stack`/`Text`/icons |
| Stateful interactive component | A handle type holding Gio state (`widget.Clickable`, `widget.Editor`, `gesture.*`), created once with a `New` constructor |

## The component recipe

1. **Define a handle type.** Hold all persistent state in it. Create it once before
   `Run` and lay it out every frame — never construct it inside the view function.

2. **Hold Gio state directly.** Gio's `widget.Clickable`, `widget.Bool`, `widget.Editor`
   and `gesture.*` types are the building blocks of interactivity. They carry their own
   event queues; just call their `Clicked`/`Update`/`Layout` methods each frame.

3. **Build the visuals out of elements** (`Box`, `Row`, `Stack`, `Text`, icons) and drop to raw
   Gio ops where needed. This is exactly how the built-in components are written.

4. **Take `styles.Styles` as a constructor argument** (optionally variadic, as `Text`
   and the built-in components do), not a chained setter, and merge defaults with
   `styles.Merge(styles.Styles{...defaults...}, userStyles)` so users can override the
   relevant properties.

5. **Chain your setters**: return the component itself from `OnClick`-style methods.

## Kit helpers

The `components/kit` package (re-exported from the root) provides the shared pieces
component authors need:

| Helper | Description |
| --- | --- |
| `kit.Scale(c, f)` | Scales a color's channels by `f`, clamped. |
| `kit.Hovered(c)` | Hover shading: slightly lighter. |
| `kit.Pressed(c)` | Pressed shading: slightly darker. |
| `kit.Alpha(c, a)` | Multiplies a color's alpha. |
| `kit.HoverTracker` | Embeddable hover-tracking building block; gives a stateless element its own `OnHover`/`Hovered()`, the same way `Text`/`Divider`/`Image`/`Progress`/`Spinner` do. |
| `kit.ClampPos(win, size, pos, margin)` | Keeps an overlay panel inside a window; flips above the anchor when it would overflow the bottom edge. |
| `kit.Margin` | Default overlay-to-edge distance (8px). |
| `kit.BlockDrag(gtx, size)` / `kit.BlockDragHere(gtx)` | Keeps a clickable/draggable/scrollable component working inside a frameless, draggable window (`Config{Frameless: true, Draggable: true}`): without it, a press on the component would be hijacked into moving the window instead of reaching the component. Call once after laying the component's own visual out, passing its measured size (`BlockDrag`), or inline if a clip scope for the component's own pointer/gesture registration is already open (`BlockDragHere`) — see `components/button`, `components/slider` for each pattern. |

## Input inside a memo

A `Memo` replays its recorded drawing while nothing inside changed, including frames with
input aimed at it, so the widgets inside aren't laid out on those frames. A widget that
handles events registers an *input pass* while it's laid out: a function that handles the
same events its `Layout` does and reports whether its appearance changed. Replaying memos
run the passes and rebuild only when one returns true.

```go
func (r *Rating) Layout(gtx layout.Context) layout.Dimensions {
	r.update(gtx)
	kit.HandleInput(r.pass) // r.pass is created once, not per frame
	...
}

func (r *Rating) update(gtx layout.Context) (changed bool) {
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: r, Kinds: pointer.Press | pointer.Enter | pointer.Leave})
		if !ok {
			return changed
		}
		... // update r's state; changed = true when it looks different now
	}
}

r.pass = func(gtx layout.Context) bool { return r.update(gtx) }
```

For a widget built on a `widget.Clickable`, `kit.ClickablePass(&clickable, update)` builds
the pass for you, reporting clicks and hover or press changes. A widget that can't provide
a pass calls `kit.OpaqueInput()` instead; enclosing memos then lay it out on every frame
the pointer is over them.

Wrap the content of interactive components in `kit.Unselectable(w)`, so their labels stay
out of an enclosing `SelectionArea`'s selection.

## Rendering above everything: portals

`elements.RegisterPortal(gtx, anchor, build)` queues `build` to be laid out and painted
after the rest of the view, regardless of how deeply nested the caller is in the element
tree — the mechanism behind `Tooltip`, `Menu`'s panel and `Dropdown`'s options list. The
app drains the queue once per frame via `elements.DrainPortals(gtx)` (already wired into
`App.Run`), so a component author never calls that part themselves.

`anchor` is a window-absolute point, typically `elements.AbsolutePosition()` (the
accumulated offset at the current point in the layout traversal), optionally offset by
the trigger's own size:

```go
anchor := elements.AbsolutePosition().Add(image.Pt(0, triggerHeight+4))
elements.RegisterPortal(gtx, anchor, func() elements.Element {
	return elements.Raw(func(gtx layout.Context) layout.Dimensions {
		return myPanel(gtx)
	})
})
```

`RegisterPortal` measures `build` once on a disabled scratch context to size it, then
clamps the position inside the window with `kit.ClampPos` — the same technique `Menu`'s
own panel sizing uses.

A portal lands at the right place wherever its caller sits: inside a `Row` or `Stack`
with several children, inside a `Scroll`, or an absolutely positioned child. A flex
container only knows where each child lands after laying all of them out, so while a
child is being laid out `AbsolutePosition` returns the container's origin; once the child
is placed, its portals, semantic nodes and tracked positions are moved to match. To keep
a position for later (to open a menu at on the next click, say), use
`elements.TrackPosition(&p)`: `p` holds the exact position once the frame's layout is
done.

A custom element that positions its own children with `op.Offset` should call
`elements.PushOffset(off)` (returns a pop func) around the same child when the offset is
known before laying it out. When it's only known afterwards, record the child with
`elements.TrackChild()` and call `Place(off)` once it is:

```go
t := elements.TrackChild()
macro := op.Record(gtx.Ops)
dims := child.Layout(gtx)
call := macro.Stop()
t.End()
off := whereItGoes(dims)
t.Place(off)
defer op.Offset(off).Push(gtx.Ops).Pop()
call.Add(gtx.Ops)
```

## Worked example: a star rating

```go
package main

import (
	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/giode/components/icon"
	"github.com/mehran9675/giode/components/kit"
	"github.com/mehran9675/giode/elements"
	"github.com/mehran9675/giode/styles"
)

var gold = "#fac815"

// Rating is a custom stateful component: five clickable stars.
type Rating struct {
	value    int
	stars    [5]widget.Clickable
	onChange func(v int)
	st       styles.Styles
}

func NewRating() *Rating {
	return &Rating{value: 3}
}

func (r *Rating) Value() int { return r.value }

func (r *Rating) OnChange(fn func(v int)) *Rating {
	r.onChange = fn
	return r
}

func (r *Rating) Styles(st styles.Styles) *Rating {
	r.st = st
	return r
}

// Layout implements elements.Element, so a *Rating can be placed
// anywhere an element is expected.
func (r *Rating) Layout(gtx layout.Context) layout.Dimensions {
	// Defaults first, user styles on top.
	st := styles.Merge(styles.Styles{Gap: 4, Color: gold}, r.st)

	var kids []elements.Element
	for i := range r.stars {
		kids = append(kids, &starEl{rating: r, index: i, st: st})
	}
	return elements.Row(st, kids...).Layout(gtx)
}

// starEl is a stateless element for one star; the rating owns the
// clickable state.
type starEl struct {
	rating *Rating
	index  int
	st     styles.Styles
}

func (s *starEl) Layout(gtx layout.Context) layout.Dimensions {
	click := &s.rating.stars[s.index]
	if click.Clicked(gtx) {
		s.rating.value = s.index + 1
		if s.rating.onChange != nil {
			s.rating.onChange(s.rating.value)
		}
	}
	return click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		col := s.st.Color
		if s.index >= s.rating.value {
			col = kit.Alpha(col, 0.25) // unselected stars are dim
		}
		return starIcon.Icon(styles.Styles{Color: col, Width: 24, Height: 24}).Layout(gtx) // starIcon: an icon.Path or loaded SVG
	})
}
```

Usage:

```go
rating := NewRating().OnChange(func(v int) { log.Printf("rated %d", v) })

app.Run(func() giode.Element {
	return giode.Box(giode.Styles{Justify: giode.JustifyCenter, Width: -1, Height: -1}, rating)
})
```

The same recipe applies to any component: an editor-based field holds a
`widget.Editor`, a dropdown holds a `widget.Clickable` plus `kit.ClampPos` for its panel,
a toggle holds a `widget.Bool`.

## Publishing your component

- Keep the handle type and its constructor in a package of your own; import Giode's
  `elements`, `styles` and `kit`.
- To mimic the built-ins, re-export it from the root `giode` package of a fork or wrap
  your own package in a small facade.
- Overlays (menus, dropdowns, popovers, tooltips) should render their popup content via
  `elements.RegisterPortal` rather than painting it in place — see "Rendering above
  everything: portals" above — so it renders above the rest of the UI regardless of
  where the component sits in the tree.

See [architecture.md](architecture.md) for how the built-in components are organized.

## Touch input

`elements.TouchInput()` (also `giode.TouchInput()`) reports whether the user is on a touch
screen: it starts on for Android and iOS, and elsewhere follows the last press, so a
laptop with a touch screen switches as the user moves between finger and mouse. When it
changes, every `Memo` rebuilds.

`elements.MinTouchTarget(gtx, w)` lays out `w` and, while `TouchInput` is on, grows the
returned size to at least 48x48dp with `w` centered. Wrap a control's visual in it inside
the widget that takes its clicks, so the hit area grows while the visual doesn't:

```go
dims := c.clickable.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
	return elements.MinTouchTarget(gtx, c.layoutVisual)
})
```

`Button`, `IconButton` and `Checkbox` use it.
