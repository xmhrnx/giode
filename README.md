# Giode

Giode is a UI library for Go built on [Gio](https://gioui.org). You describe the interface
with composable functions, style it with CSS-like properties and wire it up with plain Go
callbacks. There's no webview, it runs at native speed, and raw Gio stays reachable
everywhere.

It runs on Windows, macOS and Linux, and on Android, where giode also handles the parts of
a phone app that Gio leaves to you: the Back button, touch targets, the system pickers and
share sheet, the system bars and the app going to the background.

```go
package main

import (
	"fmt"

	"github.com/mehran9675/giode"
)

func main() {
	giode.UseGofont()

	count := 0
	btn := giode.Button("Click me", giode.Styles{Background: "#3b82f6", Color: "#ffffff", BorderRadius: 8}).
		OnClick(func() { count++ })

	app := giode.New(giode.Config{Title: "Hello", Resizable: true, Background: "theme.background"})
	app.Run(func() giode.Element {
		return giode.Stack(giode.Styles{Align: giode.AlignCenter, Justify: giode.JustifyCenter, Width: -1, Height: -1},
			giode.Text(fmt.Sprintf("Clicks: %d", count), giode.Styles{Color: "theme.text"}),
			btn,
		)
	})
}
```

## Install

```sh
go get github.com/mehran9675/giode
```

Requires Go 1.26 or later. On Linux, Gio needs the usual development packages (Wayland,
X11, EGL, Vulkan headers); see [Gio's install guide](https://gioui.org/doc/install).

The [giode CLI](https://github.com/mehran9675/giode-cli) creates projects and builds and
runs them on every platform:

```sh
go install github.com/mehran9675/giode-cli/cmd/giode@latest
giode new myapp && cd myapp && giode run
```

Mobile builds can also use Gio's `gogio` tool directly; see [docs/mobile.md](docs/mobile.md).

## Features

- **Elements and styling**: `Box`, `Row`, `Stack`, `Text`, `Table`, `Divider`, `Raw`, with
  40+ CSS-like properties (flex layout, gaps, padding, borders, gradients, shadows,
  transforms, transitions) in dp, and `OnHover`/`OnClick` on any box.
- **Components**: Button, IconButton, the form controls under `Input` (Text, Checkbox,
  Slider, Select, AutoComplete), Tabs, Scroll, VirtualList, TreeView, Dialog, Drawer, Menu
  with nested submenus, Tooltip, Toaster, Accordion, Pagination, AppShell, TitleBar for
  frameless windows, Transition, Progress, Spinner, icons and images.
- **Responsive layouts**: `Responsive` breakpoints in dp, with `giode.Compact()` telling
  any component inside which layout is active.
- **Theming**: a light/dark `Mode` and a `Palette` of semantic colors; `"theme.*"` color
  tokens follow `SetMode`/`SetPalette` everywhere, the Android system bars included.
- **State and routing**: `State[T]`, an observable value safe to use from any goroutine;
  a `Router` with history, links and the Back button.
- **Window**: size bounds, frameless and draggable windows, fullscreen, start state, icon,
  single-instance lock, clipboard, file drop, lifecycle callbacks and runtime actions.
- **Mobile**:
  - Back and Escape close the topmost dialog, drawer, menu or picker, then go back through
    the router; `HandleBack` adds your own handlers.
  - Touch-friendly defaults: 48dp touch targets, long-press tooltips, taps that land
    anywhere in a text field, on-screen keyboard types, and pickers that open as a bottom
    sheet on narrow screens.
  - `OpenFile`/`SaveFile` through the system picker, `Share` through the share sheet, and
    `OnBackground`/`OnForeground` for saving before Android pauses the app.
- **Performance**: `Memo` replays unchanged parts of the UI from cache, so large views stay
  cheap to redraw.
- **Automation**: a built-in [MCP](https://modelcontextprotocol.io) server lets AI agents
  inspect, click, type into, scroll, resize and screenshot the running app, and read its
  frame times.
- **Extensible**: write your own components with the same kit the built-ins use, or drop to
  raw Gio with `Raw`, `RunRaw` and `App.Window()`.

## Examples

```sh
go run ./examples/hello       # a minimal app
go run ./examples/showcase    # every element and component
go run ./examples/workbench   # a file browser: TreeView, VirtualList, file drop, toasts
go run ./examples/mcp         # the MCP server
```

`titlebar`, `transitions` and `custom` show frameless windows, animations and custom
components.

## Documentation

The full documentation is in [`docs/`](docs/index.md), also published as a website:

- [Getting started](docs/getting-started.md)
- [Window configuration](docs/window.md): config fields, files, sharing, Back button,
  background and foreground
- [Mobile (Android)](docs/mobile.md): building with gogio, permissions, storage, testing
  phone layouts on a desktop
- [Elements](docs/elements.md) and [style properties](docs/styles.md)
- [Components](docs/components/README.md)
- [Global state](docs/state.md), [routing](docs/routing.md) and
  [threading](docs/threading.md)
- [Rendering performance](docs/performance.md)
- [MCP server](docs/mcp.md)
- [Extending the library](docs/extending.md), [architecture](docs/architecture.md) and
  [low-level access](docs/low-level.md)

## Agent skill

[`skills/giode`](skills/giode) is a skill for AI coding agents (Claude Code and others that
read `SKILL.md`): Giode's mental model, the CLI, code organization rules and the gotchas,
with the documentation bundled in `references/`. Install it for every project with:

```sh
mkdir -p ~/.claude/skills && cp -r skills/giode ~/.claude/skills/
```

`go generate` in the repository root refreshes `references/` from `docs/`.

## License

Apache License 2.0; see [LICENSE](LICENSE).
