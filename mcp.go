package giode

import (
	"context"
	"time"

	"gioui.org/f32"
	"gioui.org/io/event"
	"github.com/xmhrnx/giode/internal/mcpbridge"
	"github.com/xmhrnx/giode/internal/platform"
	"github.com/xmhrnx/giode/internal/platform/filedrag"
	"github.com/xmhrnx/giode/internal/platform/giopatch"
	"github.com/xmhrnx/giode/mcp"
)

// MCPOptions configures the built-in MCP server enabled with
// Config.MCP. The zero value serves Streamable HTTP on
// 127.0.0.1:7600 at /mcp with every built-in tool.
type MCPOptions struct {
	// Addr is the HTTP listen address, "127.0.0.1:7600" by default.
	// Host and Port, when set, replace its parts; port 0 in Addr
	// ("127.0.0.1:0") picks any free port (see App.MCPAddress). At run
	// time the GIODE_MCP_ADDR, GIODE_MCP_HOST and GIODE_MCP_PORT
	// environment variables override all three. Binding beyond
	// loopback ("0.0.0.0") exposes the app to the network: set Token
	// when doing so.
	Addr string
	// Host is the address to listen on: "127.0.0.1" by default,
	// "0.0.0.0" for every interface.
	Host string
	// Port is the TCP port, 7600 by default.
	Port int
	// Token, when set, is required from clients as a bearer token.
	Token string
	// Stdio serves the protocol on stdin/stdout instead of HTTP, for
	// MCP clients that launch the app as a subprocess. Nothing else
	// may write to stdout then.
	Stdio bool
	// Name and Version identify the server; Name defaults to the
	// window title.
	Name, Version string
	// Instructions are sent to clients to explain the app; a default
	// describing the built-in tools is used when empty.
	Instructions string
	// ReadOnly registers only the tools that observe the app (UI
	// snapshot, screenshot, state reads), none that act on it.
	ReadOnly bool
	// DisableBuiltinTools registers no built-in tools, leaving only
	// those added with App.MCPTool.
	DisableBuiltinTools bool
	// AllowedOrigins lists extra browser origins allowed to call the
	// HTTP endpoint (localhost origins always are).
	AllowedOrigins []string
	// Timeout bounds how long a tool waits for the UI thread. Defaults
	// to 5 seconds.
	Timeout time.Duration
}

// MCPAddress returns the address the MCP server listens on
// ("127.0.0.1:7600"), once it does; "" before that, without Config.MCP,
// or on stdio. With port 0 it's the port the system picked.
func (a *App) MCPAddress() string {
	if p := a.mcpBound.Load(); p != nil {
		return *p
	}
	return ""
}

// MCPTool registers a custom tool with the app's MCP server. Use
// mcp.Func to build one from a typed Go function:
//
//	app.MCPTool(mcp.Func("add_todo", "Adds a todo item.",
//		func(ctx context.Context, in struct {
//			Title string `json:"title" description:"The todo's title"`
//		}) (any, error) {
//			todos.Update(func(t []string) []string { return append(t, in.Title) })
//			return "added", nil
//		}))
//
// Handlers run on server goroutines: touch the UI through State (which
// is safe from any goroutine) or App.RunOnUI. It's a no-op unless
// Config.MCP is set.
func (a *App) MCPTool(t mcp.Tool) {
	if a.mcp != nil {
		a.mcp.Server().AddTool(t)
	}
}

// MCPServer returns the app's MCP server for full control, or nil when
// Config.MCP is off.
func (a *App) MCPServer() *mcp.Server {
	if a.mcp == nil {
		return nil
	}
	return a.mcp.Server()
}

func newMCPBridge(a *App, opts MCPOptions) *mcpbridge.Bridge {
	return mcpbridge.New(mcpHost{a}, mcpbridge.Options(opts))
}

// mcpHost gives the MCP bridge what it needs of the App.
type mcpHost struct{ *App }

func (h mcpHost) Logf(level platform.Level, format string, args ...any) {
	h.logf(LogLevel(level), format, args...)
}
func (h mcpHost) SetBound(addr string) { h.mcpBound.Store(&addr) }
func (h mcpHost) Invalidate()          { h.invalidate() }
func (h mcpHost) Inject(ctx context.Context, e event.Event) error {
	return injectEvent(ctx, h.App, e)
}
func (h mcpHost) FrameStats() mcpbridge.FrameStats {
	st := h.App.FrameStats()
	t := func(ft FrameTiming) mcpbridge.FrameTiming { return mcpbridge.FrameTiming(ft) }
	return mcpbridge.FrameStats{Frames: st.Frames, Build: t(st.Build), Render: t(st.Render), Interval: t(st.Interval), FPS: st.FPS}
}
func (h mcpHost) FileDragWanted() bool { return h.fileDragWanted() }
func (h mcpHost) FileDragging() bool {
	_, ok := h.App.FileDrag()
	return ok
}
func (h mcpHost) FileDrag(kind filedrag.Kind, pos f32.Point) {
	h.deliverFileDrag(FileDragKind(kind), pos)
}
func (h mcpHost) FileDrop(paths []string, pos f32.Point) { h.deliverFileDrop(paths, pos) }

// injectEvent delivers e to the window's input through giode's patched
// Gio. It must not run on the UI goroutine.
func injectEvent(ctx context.Context, a *App, e event.Event) error {
	return giopatch.Inject(ctx, a.window, e)
}

// injectsKeys reports whether injectEvent works.
const injectsKeys = giopatch.InjectsEvents

// inputEpoch is when the app started, the time base of injected events.
var inputEpoch = mcpbridge.InputEpoch
