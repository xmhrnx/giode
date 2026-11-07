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

The app registers the right-click area and renders the menu on top automatically. A
right-click on an element with its own menu (see below) opens that one instead.

## On one element: Context

Every element and component has `Context(menu)`: a right-click on it, or a long-press on a
touch screen, opens the menu at the pointer, wherever the element sits in the layout.

```go
rowMenu := giode.Menu().
	Item("Rename", rename).
	Item("Delete", remove)

row := giode.Box(giode.Styles{Padding: giode.UniformInset(8)}, giode.Text(name))
row.Context(rowMenu)
```

That's all: the menu doesn't need to be laid out anywhere, the frame draws it while it's
open. The element still gets its own presses, so a `Pressable` with a context menu still
clicks. With elements inside each other, the innermost one with a menu opens its own: a
menu on a list and another on one of its rows open on the row for the row's, elsewhere on
the list for the list's. One menu can be anchored to many elements; to know which one
opened it, set a variable from the element instead (see `OpenAtClick` below).
`Context(nil)` removes it.

Elements in a view are built every frame, so call `Context` where the element is built.
Dialog, Sheet, Drawer, Menu and Toaster, which paint over the window, don't have one.

## Manual control

```go
menu.OpenAtClick(e)       // where a ClickEvent happened
menu.OpenAt(f32.Pt(x, y)) // window pixel coordinates
menu.Toggle(f32.Pt(x, y))
menu.Close()
menu.Opened()
```

Without `SetContextMenu`, lay the menu out anywhere in the view: it takes no space there
(like `Dialog`, its siblings keep their layout), its panel paints above everything else via
a portal (see [extending.md](../extending.md#rendering-above-everything-portals)), and a
press anywhere in the window outside it closes it. `OpenAt` takes window coordinates and
the panel opens exactly there, whatever containers the menu sits in. To open a menu where a
row or a tab was right-clicked (or long-pressed), pass the click to `OpenAtClick`, which
needs no Gio import:

```go
tabMenu := giode.Menu().
	Item("Close", func() { closeTab(current) }).
	Item("Close others", func() { closeOthers(current) }).
	Separator().
	Item("Move left", func() { moveTab(current, -1) }).
	Item("Move right", func() { moveTab(current, +1) })

tab := giode.Pressable(content).
	OnSecondaryClick(func(e giode.ClickEvent) { current = id; tabMenu.OpenAtClick(e) }).
	OnLongPress(func(e giode.ClickEvent) { current = id; tabMenu.OpenAtClick(e) })
```

A keyboard click (Enter or Space on the focused element) opens it below the element.
Use this when the menu's actions depend on which element opened it; otherwise `Context`
is simpler.

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
| `Shortcut(spec string) *Menu` | Shows a shortcut right-aligned on the last item added, written the platform's way: `Item("Open File…", open).Shortcut("Mod+O")` shows `⌘O` on macOS and `Ctrl+O` elsewhere. Display only; register the key with `giode.Shortcuts` (see [window.md](../window.md#keyboard-shortcuts)). |
| `DisabledItem(label string) *Menu` | Grayed-out item. |
| `Separator() *Menu` | Horizontal line. |
| `SubMenu(label string, submenu *Menu) *Menu` | Item that reveals submenu on hover. |
| `Trigger(el Element) Element` | Wraps el; clicking it opens the menu anchored below. |
| `el.Context(m)` | On any element or component: a right-click or long-press on el opens m at the pointer. |
| `OpenAtClick(e ClickEvent)` | Opens the menu where the click happened: at the pointer, or below the element for a keyboard click. |
| `OpenAt(p f32.Point)` / `Toggle(p f32.Point)` | Opens (or toggles) the menu at a window position in pixels. |

Styles: `Background`, `BorderRadius`, `Padding`, `Color` (item text).

When any item has an icon, a check or a radio dot, every row keeps that leading column, so
the labels line up. Panels are as wide as their widest row.

## Behavior

- Opens at the cursor position (or below the `Trigger` element), measured and clamped to
  the window with an 8px margin.
- A menu (or submenu) taller than the window scrolls: with the mouse wheel, a touch drag,
  or the keyboard, and a thin indicator shows the position.
- The keyboard works while a menu is open: Up and Down move the highlight (skipping
  separators and disabled items, wrapping around, and scrolling it into view), Home and
  End jump to the ends, Right (or Enter) opens the highlighted submenu on its first item,
  Left goes back out of it (swapped right to left), and Enter or Space chooses. Moving the
  pointer takes the highlight back.
- Clicking an item runs its action and closes the menu.
- Clicking outside, pressing Escape or pressing the system Back button (Android) closes it - submenus don't have this dismissal
  layer of their own; they close with whichever ancestor menu has it.
- The dismissal layer is modal: clicks under it do not reach the UI.
