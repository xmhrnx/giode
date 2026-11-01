# MCP server

Giode can run a [Model Context Protocol](https://modelcontextprotocol.io) server inside your
app, so AI agents can see and drive it while it runs — for testing, automation, or giving an
assistant hands on your app. Turn it on with one option:

```go
app := giode.New(giode.Config{
	Title: "My app",
	MCP:   true,
})
```

The server listens on `http://127.0.0.1:7600/mcp` (Streamable HTTP). Point any MCP client
at it, for example Claude Code:

```bash
claude mcp add --transport http my-app http://127.0.0.1:7600/mcp
```

Setting the environment variable `GIODE_MCP=1` enables it in any Giode app without a code
change; `GIODE_MCP_ADDR` overrides the address, and `GIODE_MCP_HOST` and `GIODE_MCP_PORT`
its parts.

## Built-in tools

| Tool | What it does |
| --- | --- |
| `ui_snapshot` | The visible UI as nodes: id, kind, label, value, bounds, accepted actions, and the scroll area a node is in. Filter by `kind` or `query`. Nodes in a scroll area can lie outside its bounds, scrolled out of view. |
| `ui_click` | Clicks a button, tab or menu item, or toggles a checkbox. |
| `ui_type` | Replaces an input's text, optionally submitting it. |
| `ui_set_value` | Sets a checkbox (`true`/`false`), slider (0–1) or select (option or index). |
| `ui_scroll` | Scrolls a scroll area (`id`, or any node inside it; without one the main scroll area) by `dx`/`dy` pixels, or with `until` (a node id or text) until that node is fully visible, paging through virtual lists whose rows aren't laid out yet. Nested scroll areas are scrolled innermost first. |
| `ui_pointer` | Real mouse events, delivered as the platform would: `move`, `press`, `release`, `click`, `double_click` or `scroll` (`dx`/`dy`), at a node's center (`id`) or window pixels (`x`, `y`), with a `button` (`primary`, `secondary`, `tertiary`) and held `modifiers`. Press, move and release drag. For custom widgets, video surfaces and anything `ui_click` can't reach. |
| `ui_key` | Presses and releases a key in the focused element: `enter`, `tab`, `escape`, arrows, `page_down`, `f5`, a letter, or a combination such as `ctrl+shift+z` (`shortcut` is Cmd on macOS, Ctrl elsewhere); `state` sends just the press or release. Type text with `ui_type`. |
| `ui_file_drag` | Drags files over the window as from the file manager: `enter`, `move`, `leave`, or `drop` with `paths`, at a node (`id`) or window pixels (`x`, `y`). Delivers what a real drag does to `OnFileDrag`, `App.FileDrag` and `OnFileDrop`, so drop targets and their highlight can be tested; `drop` alone enters first. Works without the Gio patches. |
| `screenshot` | The window as a PNG, or with `id` just that node's area (plus `padding` pixels). With an OpenGL `Config.Underlay`, the frame is read back from the GPU so video and 3D show. |
| `frame_stats` | How long recent frames took to build and render, and the frame rate (see [performance.md](performance.md#measuring-frame-times)). `reset` starts a fresh measurement. |
| `state_list` / `state_set` | Reads and sets named states (see below). |
| `set_theme` | Switches light/dark mode. |
| `window` | Minimize, maximize, fullscreen, center, raise, or `resize` to a `width` x `height` in dp - `{"action": "resize", "width": 400, "height": 840}` tests a phone layout on a desktop. |

Actions run on the UI thread, exactly as if the user did them: handlers fire and the UI
updates. `ui_pointer` and `ui_key` need the giode CLI's Gio patches (an up to date CLI), which
`giode run` builds with; in a plain `go build` they return an error saying so.

## Your own tools

Describe a tool with a typed Go function; its input struct becomes the JSON schema:

```go
app.MCPTool(mcp.Func("add_todo", "Adds an item to the todo list.",
	func(ctx context.Context, in struct {
		Title string `json:"title" description:"The todo's title"`
		Done  bool   `json:"done,omitempty"`
	}) (any, error) {
		todos.Update(func(t []Todo) []Todo { return append(t, Todo{in.Title, in.Done}) })
		return "added", nil
	}))
```

Fields are required unless tagged `omitempty`; `description` and `enum:"a,b"` tags document
them. Return a string, any JSON-encodable value, or an `*mcp.Result` (e.g. `mcp.Image`).
Handlers run on server goroutines: change the UI through `State` (safe from any goroutine)
or `app.RunOnUI(ctx, fn)`.

## Named state

States created with `giode.NamedState` are visible to `state_list` and settable through
`state_set`:

```go
var todos = giode.NamedState("todos", []Todo{})
```

## Labels for automation

Text-less controls should get a name: `IconButton(...).Label("Delete")`,
`Input.Text(...).Label("Email")`. Custom components join the snapshot with
`elements.RecordSemantic`, and become actionable by implementing `Click()`,
`ChangeText(string)`, `Toggle()`/`Checked()`, `Change(float32)` or
`Options()`/`Choose(int)`.

## Options and security

```go
MCPOptions: giode.MCPOptions{
	Host:     "127.0.0.1",      // address to listen on; "0.0.0.0" for every interface
	Port:     7600,             // TCP port
	// or both at once: Addr: "127.0.0.1:7600" ("127.0.0.1:0" for any free port)
	Token:    "secret",         // require "Authorization: Bearer secret"
	Stdio:    false,            // serve stdin/stdout instead, for clients that launch the app
	ReadOnly: false,            // only snapshot/screenshot/state_list/frame_stats
	DisableBuiltinTools: false, // only your own tools
},
```

`app.MCPAddress()` returns the address the server is listening on once it listens, which
tells an app using port 0 which port it got. The server binds to loopback by default and rejects browser requests from non-localhost
origins (DNS-rebinding protection). Set `Token` before binding to any other interface. The
`mcp` package is independent of the UI and can serve tools on its own.
