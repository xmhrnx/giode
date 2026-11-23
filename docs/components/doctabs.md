# DocTabs

A strip of document tabs, as editors and browsers show: the app adds, removes and renames
tabs as documents open and close. Each tab has a close button and a dot for unsaved
changes; an optional "+" button follows the tabs; tabs that don't fit shrink, then scroll.
The strip shows tabs only: the app lays out the selected document under it.

```go
tabs := giode.DocTabs().
	OnSelect(func(id string) { editor.Show(id) }).
	OnClose(func(id string) {
		// Ask about unsaved changes, then:
		tabs.Remove(id)
	}).
	OnNew(newDocument)

// Opening a file:
tabs.Add(giode.DocTab{ID: path, Title: filepath.Base(path), Tooltip: path,
	Icon: giode.IconSVG(goFileIcon, giode.Styles{Color: "#00acd7"})}) // optional, before the title
tabs.Select(path)

// Editing it:
tabs.SetModified(path, true)

// In the view:
giode.Stack(giode.Styles{Width: -1, Height: -1}, tabs, editorView)
```

- **Selecting.** Clicking a tab selects it and calls `OnSelect`. `Select` selects from
  code without calling it, and scrolls the tab into view. `SelectNext(1)` and
  `SelectNext(-1)` are for Ctrl+Tab and Ctrl+Shift+Tab.
- **Closing.** The close button, a middle click, and the tab menu's Close call `OnClose`.
  The tab stays until the app calls `Remove`, so it can ask about unsaved changes first.
  Without `OnClose`, closing removes the tab. Removing the selected tab selects the one
  after it, or else the one before it, and reports it to `OnSelect`; `""` means no tabs
  are left.
- **Unsaved changes.** `SetModified` shows a dot in the close button's place. The close
  button still shows while the pointer is over the tab.
- **Tab menu.** A right-click, or a long-press on touch screens, opens Close, Close Others,
  Close to the Right and Close All. Closing several tabs calls `OnCloseTabs` with all of
  them when it's set, else `OnClose` for each. `TabMenu` adds items after these;
  `NoTabMenu` turns the menu off.
- **Overflow.** Tabs are as wide as their titles, between `TabWidth`'s minimum and maximum
  (96 and 220 dp). When they don't fit, the widest shrink first, down to the minimum; past
  that the strip scrolls with the wheel or a swipe, and the "+" stays at its end. A title
  that's cut off ends in "…". Resting the pointer on a tab shows its `Tooltip`, or else its
  whole title when it's cut off.
- **Pinned tabs.** `DocTab.Pinned`, `SetPinned(id, on)` or the tab menu's Pin keeps a tab
  at the start of the strip, before the others. A pin replaces its close button, and
  clicking the pin unpins it. A middle click doesn't close it, and Close Others, Close to
  the Right and Close All leave it open; its menu's Close still does. Dragging keeps
  pinned tabs among the pinned. `OnPin` reports the user pinning or unpinning.
- **Reordering.** Dragging a tab with the mouse moves it among the others, and `OnReorder`
  gets the new order. On touch screens a swipe scrolls the strip instead.
- **Right to left.** In right-to-left locales the tabs run from the right.
- **Automation.** Tabs, their close buttons and "+" are semantic nodes that MCP clients can
  click.

## API

| Member | Description |
| --- | --- |
| `DocTabs(st ...Styles) *DocTabsElement` | Creates an empty strip. |
| `Add(tab DocTab) *DocTabsElement` | Appends a tab, or updates the tab with that ID. |
| `Insert(i int, tab DocTab)` | Puts a tab at index `i`. |
| `Remove(id string)` | Removes a tab. |
| `SetTitle(id, title string)` | Renames a tab. |
| `SetTooltip(id, tooltip string)` | Sets the text shown when the pointer rests on the tab. |
| `SetIcon(id string, icon Element)` | Sets the element shown before the title, laid out in a 16dp square (a file type icon); nil removes it. |
| `SetModified(id string, on bool)` | Shows or hides the unsaved-changes dot. |
| `SetPinned(id string, on bool)` | Pins or unpins a tab, moving it to the end of the pinned tabs or the start of the others. |
| `OnPin(fn func(id string, pinned bool))` | Called when the user pins or unpins a tab. |
| `Indicator(c Color)` | The color of the line over the selected tab; `theme.primary` by default. |
| `Rename(id, newID string)` | Changes a tab's ID (Save As), keeping its place and selection. |
| `Move(id string, to int)` | Moves a tab to index `to`. |
| `Select(id string)` | Selects a tab without calling `OnSelect`. |
| `SelectNext(step int)` | Selects the next or previous tab, wrapping around, and calls `OnSelect`. |
| `Selected() string` | ID of the selected tab, or `""`. |
| `Tabs() []DocTab`, `Tab(id)`, `Len()`, `Index(id)` | Reading the tabs. |
| `OnSelect(fn func(id string))` | Called when the user selects a tab, or a removal moves the selection. |
| `OnClose(fn func(id string))` | Called when the user closes a tab; the app calls `Remove` to remove it. |
| `OnCloseTabs(fn func(ids []string))` | Called for Close Others, Close to the Right and Close All. |
| `OnNew(fn func())` | Shows the "+" button. |
| `OnReorder(fn func(ids []string))` | Called after a drag, with the new order. |
| `TabMenu(fn func(id string, m *MenuElement))` | Adds items to a tab's menu. |
| `NoTabMenu()` | Turns the tab menu off. |
| `Closable(on bool)` | Shows or hides the close buttons (on by default). |
| `TabWidth(min, max int)` | Narrowest and widest tab, in dp. |

`DocTab` has `ID`, `Title`, `Tooltip` and `Modified`.

```go
tabs.TabMenu(func(id string, m *giode.MenuElement) {
	m.Separator().
		Item("Copy Path", func() { app.SetClipboard(id) }).
		Item("Reveal in Folder", func() { app.Share(ctx, giode.ShareFile{Path: id}) })
})
```

## Styles

| Property | Role |
| --- | --- |
| `Background` | The strip, behind the tabs. The selected tab takes `theme.background`. |
| `Color` | The selected tab's title. The others use `theme.muted`. |
| `BorderColor` | The lines between tabs and under the strip (`theme.border`). |
| `FontSize` | Title size, 13 by default. |
| `Height` | Strip height in dp, 34 by default. |
