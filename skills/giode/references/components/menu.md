# Context menu

Right-click menu that opens at the cursor and keeps itself within the window bounds,
flipping above the cursor when it would overflow the bottom edge.

## Global right-click

```go
menu := giode.Menu(giode.Styles{}).
	Item("New", func() { ... }).
	IconItem(trashIcon, "Delete", func() { ... }). // any IconSource, see Icons
	Separator().
	DisabledItem("Locked")

app := giode.New(giode.Config{...})
app.SetContextMenu(menu) // right-click anywhere in the window opens it
```

The app registers the right-click area and renders the menu on top automatically.

## Manual control

```go
menu.OpenAt(f32.Pt(x, y)) // window pixel coordinates
menu.Toggle(f32.Pt(x, y))
menu.Close()
menu.Opened()
```

Without `SetContextMenu`, lay the menu out anywhere in the view - its panel renders via
a portal (see [extending.md](../extending.md#rendering-above-everything-portals)), so it
paints above everything else regardless of tree position.

## Dropdown menu: Trigger

Wrap any element so clicking it opens the menu anchored just below, like a "File" menu
button:

```go
trigger := giode.Button("File", giode.Styles{Background: "gray"})
wrapped := menu.Trigger(trigger)

// in the view, lay out wrapped in trigger's place, and lay menu out
// somewhere too (its panel still renders via a portal):
giode.Stack(giode.Styles{}, wrapped, menu)
```

## Nested submenus: SubMenu

`SubMenu` appends an item that reveals a nested panel, cascading to the side, when
hovered:

```go
sub := giode.Menu().
	Item("Copy", func() { ... }).
	Item("Paste", func() { ... })

menu := giode.Menu().
	Item("New", func() { ... }).
	SubMenu("More", sub)
```

`sub` is a `*Menu` built the same way as any other, and can itself contain further
`SubMenu` items for deeper nesting - but must not be registered with `SetContextMenu`,
`OpenAt`, `Toggle` or `Trigger` itself: its parent item's hover is its only way of
opening. Once hover opens it, it stays open until a sibling submenu item is hovered
instead (not the moment the pointer leaves the row), so moving into the submenu's own
panel doesn't flicker it shut. Closing an ancestor menu (`Close`, clicking outside,
Escape) closes every open submenu with it.

## API

| Member | Description |
| --- | --- |
| `Menu(st ...Styles) *Menu` | Creates an empty menu. |
| `Item(label string, action func()) *Menu` | Clickable item. |
| `IconItem(iconName, label string, action func()) *Menu` | Item with a material icon. |
| `CheckItem(label string, checked func() bool, action func()) *Menu` | Item with a check mark while `checked` reports true (read every frame). |
| `RadioItem(label string, selected func() bool, action func()) *Menu` | One of a group of choices, with a dot on the selected one. |
| `Shortcut(text string) *Menu` | Shows a shortcut right-aligned on the last item added: `Item("Open File…", open).Shortcut("Ctrl+O")`. Display only; handle the key with the `shortcut` package. |
| `DisabledItem(label string) *Menu` | Grayed-out item. |
| `Separator() *Menu` | Horizontal line. |
| `SubMenu(label string, submenu *Menu) *Menu` | Item that reveals submenu on hover. |
| `Trigger(el Element) Element` | Wraps el; clicking it opens the menu anchored below. |

Styles: `Background`, `BorderRadius`, `Padding`, `Color` (item text).

When any item has an icon, a check or a radio dot, every row keeps that leading column, so
the labels line up. Panels are as wide as their widest row.

## Behavior

- Opens at the cursor position (or below the `Trigger` element), measured and clamped to
  the window with an 8px margin.
- Clicking an item runs its action and closes the menu.
- Clicking outside, pressing Escape or pressing the system Back button (Android) closes it - submenus don't have this dismissal
  layer of their own; they close with whichever ancestor menu has it.
- The dismissal layer is modal: clicks under it do not reach the UI.
