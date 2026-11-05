package giode

import (
	"context"
	"errors"
	"fmt"
	"image"
	"net"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"gioui.org/gpu/headless"
	"gioui.org/op"
	"github.com/xmhrnx/giode/elements"
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

const defaultMCPAddr = "127.0.0.1:7600"

// mcpAddr resolves the MCP listen address from the options and the
// environment.
func mcpAddr(o MCPOptions, getenv func(string) string) (string, error) {
	addr := o.Addr
	if env := getenv("GIODE_MCP_ADDR"); env != "" {
		addr = env
	}
	if addr == "" {
		addr = defaultMCPAddr
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", fmt.Errorf("listen address %q: %w", addr, err)
	}
	if o.Host != "" {
		host = o.Host
	}
	if o.Port != 0 {
		port = strconv.Itoa(o.Port)
	}
	if env := getenv("GIODE_MCP_HOST"); env != "" {
		host = env
	}
	if env := getenv("GIODE_MCP_PORT"); env != "" {
		port = env
	}
	if n, err := strconv.Atoi(port); err != nil || n < 0 || n > 65535 {
		return "", fmt.Errorf("port %q isn't a TCP port", port)
	}
	return net.JoinHostPort(host, port), nil
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

// mcpBridge connects MCP tool handlers, which run on server
// goroutines, with the UI goroutine that owns the widget tree.
type mcpBridge struct {
	srv  *mcp.Server
	opts MCPOptions

	mu         sync.Mutex
	before     []func()                              // run before the next layout
	after      []func(ops *op.Ops, size image.Point) // run after the next frame
	nodes      []elements.SemanticNode               // last frame's semantic tree
	winSize    image.Point
	headless   *headless.Window
	headlessSz image.Point

	// An underlay's frames, read back on the render thread: offscreen
	// rendering of the UI alone would leave the underlay out.
	underlayGL  atomic.Bool // an OpenGL underlay is drawing
	wantCapture atomic.Bool // the next frame is to be read back
	captured    *image.RGBA // the frame read back, guarded by mu
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
		a.mcp.srv.AddTool(t)
	}
}

// MCPServer returns the app's MCP server for full control, or nil when
// Config.MCP is off.
func (a *App) MCPServer() *mcp.Server {
	if a.mcp == nil {
		return nil
	}
	return a.mcp.srv
}

func newMCPBridge(a *App, opts MCPOptions) *mcpBridge {
	if opts.Name == "" {
		opts.Name = a.cfg.Title
		if opts.Name == "" {
			opts.Name = "giode-app"
		}
	}
	if opts.Version == "" {
		opts.Version = "1.0.0"
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 5 * time.Second
	}
	b := &mcpBridge{srv: mcp.NewServer(opts.Name, opts.Version), opts: opts}
	b.srv.Token = opts.Token
	b.srv.AllowedOrigins = opts.AllowedOrigins
	b.srv.Instructions = opts.Instructions
	if b.srv.Instructions == "" {
		b.srv.Instructions = "This server controls a running desktop app built with Giode. " +
			"Call ui_snapshot to see the visible UI as a list of nodes with ids, then act on them with " +
			"ui_click, ui_type, ui_set_value and ui_scroll, and verify with ui_snapshot or screenshot."
	}
	if !opts.DisableBuiltinTools {
		b.registerBuiltins(a)
	}
	return b
}

// start serves the protocol until ctx is canceled.
func (b *mcpBridge) start(ctx context.Context, a *App) {
	if b.opts.Stdio {
		go func() {
			if err := b.srv.ServeStdio(ctx, os.Stdin, os.Stdout); err != nil && !errors.Is(err, context.Canceled) {
				a.logf(LogLevelError, "giode: mcp stdio: %v", err)
			}
		}()
		a.logf(LogLevelInfo, "giode: mcp server on stdio")
		return
	}
	addr, err := mcpAddr(b.opts, os.Getenv)
	if err != nil {
		a.logf(LogLevelError, "giode: mcp server: %v", err)
		return
	}
	go func() {
		err := b.srv.ListenAndServe(ctx, addr, func(bound string) {
			a.mcpBound.Store(&bound)
			a.logf(LogLevelInfo, "giode: mcp server listening on http://%s/mcp", bound)
		})
		if err != nil {
			a.logf(LogLevelError, "giode: mcp server: %v", err)
		}
	}()
}

func (b *mcpBridge) release() {
	if b.headless != nil {
		b.headless.Release()
		b.headless = nil
	}
}

// --- Built-in tools ---------------------------------------------------------
