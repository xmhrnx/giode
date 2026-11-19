# Drag and drop

Moving things around inside the app: cards between the columns of a board, rows within
a list, items from one panel to another. A `Draggable` carries a payload; a `DropTarget`
says which payloads it takes and receives them. While a drag runs, a preview of the
dragged element follows the pointer or finger.

```go
// A card the user can drag. Create it once per card (Keyed in lists).
card := giode.Draggable(task.ID, func(s giode.DragState) giode.Element {
	return giode.Stack(giode.Styles{Padding: giode.UniformInset(10), Background: "theme.background",
		BorderRadius: 6, BorderWidth: 1, BorderColor: "theme.border"},
		giode.Text(task.Title))
})

// A column that takes task IDs, as a list: drops go between its cards.
column := giode.DropTarget(func(s giode.DropState) giode.Element {
	bg := "theme.surface"
	if s.Over {
		bg = "theme.border"
	}
	var cards []giode.Element
	for i, t := range tasks[stage] {
		cards = append(cards, giode.DropListItem(i, cardFor(t)))
	}
	return giode.Stack(giode.Styles{Width: 260, Height: -1, Gap: 8, Background: bg}, cards...)
}).List(giode.DragAxisVertical).
	Accept(giode.DragAccept[TaskID]()).
	OnDrop(func(e giode.DropEvent) {
		id, _ := giode.DragPayload[TaskID](e.Payload)
		moveTask(id, stage, e.Index)
	})
```

## Starting a drag

- **Mouse.** Press and move a few dp. A press that doesn't move is still a click: the
  `Draggable`'s content keeps its own `OnClick`s, and a drag cancels them.
- **Touch.** Hold for half a second (`LongPressDuration`); a haptic tap confirms the
  pickup. A finger that moves at once scrolls instead, so cards in a `Scroll` or
  `LazyList` don't get in the way of scrolling. `TouchImmediate(true)` starts on the
  first movement, for a drag handle that isn't in the way.
- **Keyboard.** Tab to a `Draggable` and press Space or Enter to pick it up. Up and Down
  move it between the slots of a list, Left and Right to the previous or next drop
  target (a board's columns), Space or Enter drops it, Escape puts it back.

Escape cancels a pointer drag too, as does `giode.CancelDrag()`. A release where no target
takes the payload cancels it.

## The dragged element

While dragged, the `Draggable` shows faded in its place (`Source(giode.DragSourceHide)`
leaves the space empty, `DragSourceKeep` leaves it as it is), and a copy follows the
pointer, lifted with a shadow, held where it was grabbed. `Preview(func() Element)` draws
something else instead: a compact chip for a large row, a count for several items.
`LockAxis(giode.DragAxisVertical)` keeps the preview in its column, for reordering a list.

`DragState` tells the content it's being dragged (`Dragging`), hovered or focused.
`OnDragStart` and `OnDragEnd(func(giode.DragResult))` report the drag's start and end:
`Dropped` and the `Target` and `Index` it went to, or `Cancelled`.

The drag belongs to the window once it starts: the source can scroll out of view, or
leave the tree, and the drag goes on.

## Drop targets

`DropTarget(func(giode.DropState) Element)` is an area drags can be dropped on.

- `Accept(func(payload any) bool)` chooses the payloads it takes; without it, it takes
  any. `giode.DragAccept[T]()` takes payloads of type `T`.
- `OnDrop(func(giode.DropEvent))` receives a drop: the `Payload`, the `Source`
  Draggable, the list slot `Index`, the `Position` in the target, and whether it was
  made with the `Keyboard`.
- `OnEnter` and `OnLeave` report a drag it takes moving over it and away.
- `DropState` lets it restyle itself: `Active` while a drag it takes is in progress
  (to show where drops can go), `Over` while that drag is over it, and `Index`, the slot
  it would drop at.

Targets nest: the innermost one under the pointer that takes the payload gets the drop,
so a board can have a trash area over its columns, or columns inside a scrolling board.
A target inside a `Scroll`, `LazyList` or `VirtualList` takes drops only where it shows.

## Lists

`List(axis)` makes a target a list. Its items are the elements wrapped in
`giode.DropListItem(index, element)` inside it, with their indexes in the list's data. A
drop goes into the slot nearest the pointer, between two items: `Index` is 0 before the
first item and `n` after item `n-1`. A line shows the slot while the drag is over the
list (`Indicator(false)` hides it; draw your own gap at `DropState.Index` instead).

Virtualized lists work the same way: wrap each row a `LazyList` or `VirtualList` builder
returns. Only the rows on screen are laid out, and only those have slots, which is all
the pointer can reach; dragging near the list's edge scrolls it to bring more in.

The slot is an insertion position, so moving an item within its own list has to account
for its removal. `giode.MoveItem(slice, from, to)` does that; `InsertItem` and
`RemoveItem` move an item between lists:

```go
OnDrop(func(e giode.DropEvent) {
	id, _ := giode.DragPayload[TaskID](e.Payload)
	from, i := find(id)
	if from == stage {
		giode.MoveItem(tasks[stage], i, e.Index)
		return
	}
	tasks[from] = giode.RemoveItem(tasks[from], i)
	tasks[stage] = giode.InsertItem(tasks[stage], e.Index, task)
})
```

## Auto-scroll

`Scroll`, `LazyList` and `VirtualList` scroll while a drag is held near one of their edges
(within 56 dp, or a quarter of the viewport), faster the closer it is, and keep going
just past the edge. A horizontal board inside a `Scroll` scrolls sideways, and each
column's list scrolls up and down.

## Performance

Drop targets and list items register where they are every frame of a drag, so a `Memo`
holding one lays its subtree out while a drag runs and replays it again after. A board
of 500 cards in 6 `LazyList` columns lays out a drag frame in about 1.5 ms on a desktop
CPU, the same as a frame without a drag.

## Not yet

Dragging out of the app (files or text to another application) and between the app's
windows aren't supported yet. Files dragged in from the system arrive through
`Config.OnFileDrop`; see [window.md](../window.md).

`Pressable`'s `OnDragStart`, `OnDragMove` and `OnDragEnd` remain for gestures that move
something by hand, like a slider or a splitter; see [pressable.md](pressable.md).
