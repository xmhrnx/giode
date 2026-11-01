# Giode

Giode is a UI library built on [Gio](https://gioui.org) that brings retained-mode ease to
immediate-mode UI. Interfaces are described with composable functions, styled with CSS-like
properties and wired up with plain Go callbacks — no webview, full performance, and raw Gio
stays reachable everywhere.

> Giode is in **alpha**: it's used in real apps, but the API will change before 1.0.

New here? [Getting started](getting-started.md) takes you from installing the CLI to a
running app in five minutes. [Recipes](recipes.md) shows the ways to write components,
and the [FAQ](faq.md) answers the common questions.

## Features

- **Window config** — size, min/max, resizable, frameless, draggable, fullscreen, centered,
  always-on-top, per-axis size bounds, initial window state, window icon, lifecycle
  callbacks (startup, ready, shutdown, close, background, foreground) and a
  single-instance lock, plus runtime actions (minimize, maximize, center, raise, resize,
  close).
- **Mobile (Android)** — the Back button closes the topmost overlay and goes back through
  the router (`HandleBack` for your own handlers); 48dp touch targets, long-press
  tooltips, on-screen keyboard types and bottom-sheet pickers on touch screens; system
  bar colors that follow the theme; `OpenFile`/`SaveFile` through the system picker and
  `Share` through the share sheet.
- **Elements** — `Box`, `Row`, `Stack`, `Text`, `Divider`, `Table`/`TableHeader`/
  `TableRow`/`TableCell`, `Raw`, `Responsive`, most with `OnHover` for pointer-reactive
  styling.
- **Components** — Button (contained/outlined/ghost variants), IconButton, Progress,
  Spinner, Tabs, Drawer, Dialog, Scroll (with intersection-observer-style
  `OnReachBottom`/`Visible`), VirtualList, TreeView, Toaster, TitleBar, Transition,
  Pressable, Title, Image, Tooltip, Accordion, Pagination, AppShell (Header/Navbar/Aside/
  Footer/Main/Section), a Menu with a right-click trigger, a widget-anchored dropdown
  trigger and hover-nested submenus, and the form controls grouped under `Input`: Text,
  Checkbox, Slider, Select, AutoComplete.
- **Portals** — any component can render above everything else regardless of nesting
  depth via `elements.RegisterPortal` - the mechanism behind Tooltip, Menu's panel and
  Dropdown/AutoComplete's suggestion list.
- **Theming** — a `Palette` of semantic colors and a light/dark `Mode`; any `"theme.*"`
  color token (e.g. `Background: "theme.surface"`) follows `SetMode`/`SetPalette`.
- **Global state** — `State[T]`, a generic observable value readable/writable from
  anywhere, including a goroutine outside the render loop.
- **CSS-like styling** — 40+ properties, each defined in its own file, extensible by design.
- **Routing** — `NewRouter`, `Push`/`Navigate`/`Back`, `Link`.
- **Breakpoints** — `Responsive` switches views when the available width crosses a threshold
  in dp; `Compact()`/`ActiveBreakpoint()` tell components inside which one matched.
- **Performance** — `Memo` replays unchanged parts of the UI from cache.
- **Automation** — a built-in MCP server lets AI agents inspect and drive the running app.
- **Icons** — Material set, SVG path data, a full `.svg` file loader, IconVG, plain images.
- **Component kit** — create your own components with the same helpers the built-ins use.
- **Low-level escape hatches** — `Raw`, `RunRaw`, `App.Window()`.

## Documentation

- [Getting started](getting-started.md)
- [Window configuration](window.md)
- [Mobile (Android)](mobile.md)
- [Platform services](platform.md)
- [Elements](elements.md)
- [Style properties](styles.md)
- [Components](components/)
- [Global state](state.md)
- [Routing](routing.md)
- [Threading](threading.md)
- [Rendering performance](performance.md)
- [MCP server](mcp.md)
- [Extending the library](extending.md)
- [Architecture](architecture.md)
- [Low-level access](low-level.md)

## Packages

The root package `giode` re-exports the whole API for single-import ergonomics. The
underlying packages stay importable directly:

| Package | Contents |
| --- | --- |
| `styles/properties` | One file per CSS-like property |
| `styles` | The `Styles` aggregate, `Merge`, `Hex` |
| `elements` | Stateless building blocks |
| `components/*` | Stateful components, one directory each |
| `components/kit` | Helpers for custom components |
| `state` | The generic `State[T]` global-state primitive |
| `router` | Navigation |
| `fonts` | Font/shaper registry |

## About this documentation

The pages you are reading are Markdown files in this folder, rendered by a Vite site in
the `web/` directory. See [README.md](README.md) for the structure and how to add pages.

