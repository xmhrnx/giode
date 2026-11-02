# Components

Stateful components are created once and laid out every frame. They are built on the
stateless elements.

| Component | Constructor | State |
| --- | --- | --- |
| Button | [button.md](button.md) | clickable |
| Input (text field) | [input.md](input.md) | text |
| Input (checkbox) | [input.md](input.md) | bool |
| Input (slider) | [input.md](input.md) | value |
| Input (select) | [input.md](input.md) | selection |
| Input (autocomplete) | [input.md](input.md) | text, suggestions |
| Tabs | [tabs.md](tabs.md) | selection |
| Scroll | [scroll.md](scroll.md) | offset |
| Drawer | [drawer.md](drawer.md) | external `*bool` |
| Dialog | [dialog.md](dialog.md) | external `*bool` |
| Menu | [menu.md](menu.md) | items, open |
| Tooltip | [tooltip.md](tooltip.md) | hover |
| TreeView | [treeview.md](treeview.md) | selection, expansion |
| VirtualList | [virtuallist.md](virtuallist.md) | visible range, focused index |
| LazyList | [lazylist.md](lazylist.md) | anchor row, measured heights |
| Toaster | [toaster.md](toaster.md) | toasts showing |
| Pressable | [pressable.md](pressable.md) | press state, click events |
| TitleBar | [titlebar.md](titlebar.md) | window state, buttons |
| Transition | [transition.md](transition.md) | visibility, progress |
| Progress | [progress.md](progress.md) | stateless |
| Spinner | [spinner.md](spinner.md) | stateless, animated |
| Image | [image.md](image.md) | stateless |
| Icons | [icons.md](icons.md) | stateless |
| Title | [title.md](title.md) | stateless |
| AppShell | [appshell.md](appshell.md) | stateless (Section wraps Scroll) |
| Accordion | [accordion.md](accordion.md) | expanded sections |
| Pagination | [pagination.md](pagination.md) | current page |
| Swipeable | [swipeable.md](swipeable.md) | offset, open side |
| Zoomable | [zoom.md](zoom.md) | scale, offset |
| Sheet | [sheet.md](sheet.md) | external `*bool`, detent |
| Input (switch) | [controls.md](controls.md#switch) | bool |
| Input (segmented) | [controls.md](controls.md#segmented) | selection |
| Chip, Badge, Avatar | [controls.md](controls.md) | stateless (a clickable Chip holds press state) |
| Table | [table.md](table.md) | stateless |

Common patterns:

- **Constructors take styles last**: `giode.Button("Go", giode.Styles{...})`,
  `giode.Input.Text("Name", giode.Styles{...})`, `giode.Scroll(content, styles)`.
- **Callbacks**: handlers are plain Go closures and may capture anything; chainable
  setters return the component.
- **Styling**: every component accepts `Styles`; see the individual docs for which
  properties are honored.
