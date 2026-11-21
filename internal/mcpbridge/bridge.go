// Package mcpbridge is the built-in MCP server of giode apps (Config.MCP):
// tools that let an agent see and drive the running UI.
package mcpbridge

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

	"gioui.org/f32"
	"gioui.org/gpu/headless"
	"gioui.org/io/event"
	"gioui.org/op"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/internal/platform"
	"github.com/xmhrnx/giode/internal/platform/filedrag"
	"github.com/xmhrnx/giode/mcp"
)

// Options configures the bridge, as giode.MCPOptions: The zero value serves Streamable HTTP on
// 127.0.0.1:7600 at /mcp with every built-in tool.
type Options struct {
	// Addr is the HTTP listen address, "127.0.0.1:7600" by default.
	// Host and Port, when set, replace its parts; port 0 in Addr
	// ("127.0.0.1:0") picks any free port (see giode.App.MCPAddress). At run
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
	// those added with giode.App.MCPTool.
	DisableBuiltinTools bool
	// AllowedOrigins lists extra browser origins allowed to call the
	// HTTP endpoint (localhost origins always are).
	AllowedOrigins []string
	// Timeout bounds how long a tool waits for the UI thread. Defaults
	// to 5 seconds.
	Timeout time.Duration
}

const defaultMCPAddr = "127.0.0.1:7600"

// listenAddr resolves the MCP listen address from the options and the
// environment.
func listenAddr(o Options, getenv func(string) string) (string, error) {
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

// Bridge connects MCP tool handlers, which run on server
// goroutines, with the UI goroutine that owns the widget tree.
type Bridge struct {
	h    Host
	srv  *mcp.Server
	opts Options

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

// Host is what the bridge needs of the app.
type Host interface {
	Title() string
	Logf(level platform.Level, format string, args ...any)
	// SetBound records the address the server listens on.
	SetBound(addr string)
	// RunOnUI runs fn on the UI goroutine and waits for it.
	RunOnUI(ctx context.Context, fn func()) error
	// Invalidate asks for a frame.
	Invalidate()
	// Inject delivers an input event to the window.
	Inject(ctx context.Context, e event.Event) error
	// The window, for the window tool.
	SetSize(w, h int)
	Minimize()
	Maximize()
	Unmaximize()
	SetFullscreen(on bool)
	Center()
	Raise()
	// Frame timing, for frame_stats.
	FrameStats() FrameStats
	ResetFrameStats()
	// File drags, for ui_drop_files: whether the app takes them,
	// whether one is over the window, and delivering them.
	FileDragWanted() bool
	FileDragging() bool
	FileDrag(kind filedrag.Kind, pos f32.Point)
	FileDrop(paths []string, pos f32.Point)
}

// FrameStats and FrameTiming are giode.FrameStats and FrameTiming.
type FrameStats struct {
	Frames                  int
	Build, Render, Interval FrameTiming
	FPS                     float64
}

type FrameTiming struct {
	Avg, P95, Max time.Duration
}

// Server is the MCP server.
func (b *Bridge) Server() *mcp.Server { return b.srv }

// New makes the bridge and registers the built-in tools.
func New(h Host, opts Options) *Bridge {
	if opts.Name == "" {
		opts.Name = h.Title()
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
	b := &Bridge{h: h, srv: mcp.NewServer(opts.Name, opts.Version), opts: opts}
	b.srv.Token = opts.Token
	b.srv.AllowedOrigins = opts.AllowedOrigins
	b.srv.Instructions = opts.Instructions
	if b.srv.Instructions == "" {
		b.srv.Instructions = "This server controls a running desktop app built with Giode. " +
			"Call ui_snapshot to see the visible UI as a list of nodes with ids, then act on them with " +
			"ui_click, ui_type, ui_set_value and ui_scroll, and verify with ui_snapshot or screenshot."
	}
	if !opts.DisableBuiltinTools {
		b.registerBuiltins()
	}
	return b
}

// Start serves the protocol until ctx is canceled.
func (b *Bridge) Start(ctx context.Context) {
	a := b.h
	if b.opts.Stdio {
		go func() {
			if err := b.srv.ServeStdio(ctx, os.Stdin, os.Stdout); err != nil && !errors.Is(err, context.Canceled) {
				a.Logf(platform.Error, "giode: mcp stdio: %v", err)
			}
		}()
		a.Logf(platform.Info, "giode: mcp server on stdio")
		return
	}
	addr, err := listenAddr(b.opts, os.Getenv)
	if err != nil {
		a.Logf(platform.Error, "giode: mcp server: %v", err)
		return
	}
	go func() {
		err := b.srv.ListenAndServe(ctx, addr, func(bound string) {
			a.SetBound(bound)
			a.Logf(platform.Info, "giode: mcp server listening on http://%s/mcp", bound)
		})
		if err != nil {
			a.Logf(platform.Error, "giode: mcp server: %v", err)
		}
	}()
}

// Release frees the offscreen renderer; on the UI goroutine.
func (b *Bridge) Release() {
	if b.headless != nil {
		b.headless.Release()
		b.headless = nil
	}
}
