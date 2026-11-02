package giode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gioui.org/gpu/headless"
	"gioui.org/op"

	"github.com/mehran9675/giode/elements"
	"github.com/mehran9675/giode/mcp"
	"github.com/mehran9675/giode/state"
	"github.com/mehran9675/giode/styles/properties"
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

// captureUnderlay reads back a frame drawn over an underlay when a
// screenshot asked for one. It runs on the render thread.
func (b *mcpBridge) captureUnderlay(t GPUTarget) {
	b.underlayGL.Store(t.API == GPUOpenGL)
	if t.API != GPUOpenGL || !b.wantCapture.Swap(false) {
		return
	}
	img := readFramebuffer(t)
	b.mu.Lock()
	b.captured = img
	b.mu.Unlock()
}

// takeCapture returns the frame captureUnderlay read back, if any.
func (b *mcpBridge) takeCapture() *image.RGBA {
	b.mu.Lock()
	defer b.mu.Unlock()
	img := b.captured
	b.captured = nil
	return img
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

// Post queues fn to run on the UI goroutine before the next frame and
// requests that frame, without waiting. While the window is hidden or
// minimized, and there are no frames, it runs as soon as the UI
// goroutine wakes up. It is safe to call from any
// goroutine: use it to apply results from background work (a network
// response, a file read) to data the view reads, so that data never
// needs locks. Calls run in the order posted.
func (a *App) Post(fn func()) {
	a.uiMu.Lock()
	a.uiCalls = append(a.uiCalls, fn)
	a.uiMu.Unlock()
	a.window.Invalidate()
}

// RunOnUI runs fn on the UI goroutine before the next frame is laid
// out (or straight away while the window is hidden) and waits for it,
// so code running elsewhere (an MCP tool, a network callback) can
// safely drive stateful components. It fails if the UI goroutine
// doesn't run it before ctx is done.
func (a *App) RunOnUI(ctx context.Context, fn func()) error {
	done := make(chan struct{})
	a.uiMu.Lock()
	a.uiCalls = append(a.uiCalls, func() { fn(); close(done) })
	a.uiMu.Unlock()
	a.window.Invalidate()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("giode: UI thread didn't run the call (window closed or busy?): %w", ctx.Err())
	}
}

// runUICalls runs the queued Post and RunOnUI calls, reporting whether
// there were any. The frame loop calls it before laying out, and on
// the wakeups Invalidate causes when no frame comes (a hidden or
// minimized window), so tray menus and other callers still reach the
// UI goroutine.
func (a *App) runUICalls() bool {
	a.uiMu.Lock()
	calls := a.uiCalls
	a.uiCalls = nil
	a.uiMu.Unlock()
	if len(calls) == 0 {
		return false
	}
	for _, fn := range calls {
		fn()
	}
	// Calls usually drive components programmatically; make sure
	// cached subtrees pick that up.
	elements.InvalidateCaches()
	return true
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

// frameDone records the frame's semantic tree and runs after-frame
// callbacks; the frame loop calls it after presenting a frame.
func (b *mcpBridge) frameDone(nodes []elements.SemanticNode, ops *op.Ops, size image.Point) {
	b.mu.Lock()
	b.nodes = nodes
	b.winSize = size
	after := b.after
	b.after = nil
	b.mu.Unlock()
	for _, fn := range after {
		fn(ops, size)
	}
}

// afterFrame queues fn to run on the UI goroutine right after the next
// frame, requests that frame and waits.
func (b *mcpBridge) afterFrame(ctx context.Context, a *App, fn func(ops *op.Ops, size image.Point)) error {
	done := make(chan struct{})
	b.mu.Lock()
	b.after = append(b.after, func(ops *op.Ops, size image.Point) { fn(ops, size); close(done) })
	b.mu.Unlock()
	a.window.Invalidate()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("giode: no frame was drawn (window minimized or closed?): %w", ctx.Err())
	}
}

// snapshot waits for a fresh frame and returns its semantic tree.
func (b *mcpBridge) snapshot(ctx context.Context, a *App) ([]elements.SemanticNode, image.Point, error) {
	if err := b.afterFrame(ctx, a, func(*op.Ops, image.Point) {}); err != nil {
		return nil, image.Point{}, err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.nodes, b.winSize, nil
}

// node finds a node of the last frame by id.
func (b *mcpBridge) node(id string) (elements.SemanticNode, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, n := range b.nodes {
		if n.ID == id {
			return n, true
		}
	}
	return elements.SemanticNode{}, false
}

// screenshot renders the next frame's ops offscreen. It runs on the UI
// goroutine.
func (b *mcpBridge) screenshot(ops *op.Ops, size image.Point) (*image.RGBA, error) {
	if size.X <= 0 || size.Y <= 0 {
		return nil, errors.New("window has no size")
	}
	if b.headless == nil || b.headlessSz != size {
		if b.headless != nil {
			b.headless.Release()
		}
		w, err := headless.NewWindow(size.X, size.Y)
		if err != nil {
			b.headless = nil
			return nil, fmt.Errorf("offscreen renderer unavailable: %w", err)
		}
		b.headless, b.headlessSz = w, size
	}
	if err := b.headless.Frame(ops); err != nil {
		return nil, err
	}
	img := image.NewRGBA(image.Rectangle{Max: size})
	if err := b.headless.Screenshot(img); err != nil {
		return nil, err
	}
	return img, nil
}

func encodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (b *mcpBridge) release() {
	if b.headless != nil {
		b.headless.Release()
		b.headless = nil
	}
}

// --- Built-in tools ---------------------------------------------------------

// snapshotNode is a node as reported to clients.
type snapshotNode struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Label    string `json:"label,omitempty"`
	Value    string `json:"value,omitempty"`
	Bounds   [4]int `json:"bounds"`
	Disabled bool   `json:"disabled,omitempty"`
	Actions  string `json:"actions,omitempty"`
	Scroll   string `json:"scroll,omitempty"`
}

// Interfaces stateful components implement for automation.
type (
	clicker     interface{ Click() }
	textChanger interface{ ChangeText(string) }
	textSetter  interface{ SetText(string) }
	submitter   interface{ SubmitText() }
	toggler     interface {
		Checked() bool
		Toggle()
	}
	valueChanger  interface{ Change(float32) }
	optionChooser interface {
		Options() []string
		Choose(int)
	}
)

func actionsOf(target any) string {
	var acts []string
	if _, ok := target.(clicker); ok {
		acts = append(acts, "click")
	}
	_, tc := target.(textChanger)
	_, ts := target.(textSetter)
	if tc || ts {
		acts = append(acts, "type")
	}
	if _, ok := target.(toggler); ok {
		acts = append(acts, "set_value(bool)")
	}
	if _, ok := target.(valueChanger); ok {
		acts = append(acts, "set_value(0-1)")
	}
	if _, ok := target.(optionChooser); ok {
		acts = append(acts, "set_value(option)")
	}
	return strings.Join(acts, ",")
}

func (b *mcpBridge) registerBuiltins(a *App) {
	ctxFor := func(ctx context.Context) (context.Context, context.CancelFunc) {
		return context.WithTimeout(ctx, b.opts.Timeout)
	}

	snapshotTool := mcp.Func("ui_snapshot", "Returns the visible UI of the app as a flat list of nodes: "+
		"text runs and controls with an id, kind, label, current value, window-relative bounds [x, y, width, height] "+
		"in pixels and the actions they accept. Use the ids with ui_click, ui_type and ui_set_value. "+
		"Nodes inside a scroll area name it in scroll; they can lie outside its bounds, scrolled out of view (see ui_scroll).",
		func(ctx context.Context, in struct {
			Kind  string `json:"kind,omitempty" description:"Only return nodes of this kind (text, button, input, checkbox, slider, select, tab, menuitem, image, scroll)"`
			Query string `json:"query,omitempty" description:"Only return nodes whose label or value contains this text (case-insensitive)"`
		}) (any, error) {
			ctx, cancel := ctxFor(ctx)
			defer cancel()
			nodes, size, err := b.snapshot(ctx, a)
			if err != nil {
				return nil, err
			}
			q := strings.ToLower(in.Query)
			out := make([]snapshotNode, 0, len(nodes))
			for _, n := range nodes {
				if in.Kind != "" && string(n.Kind) != in.Kind {
					continue
				}
				if q != "" && !strings.Contains(strings.ToLower(n.Label), q) && !strings.Contains(strings.ToLower(n.Value), q) {
					continue
				}
				out = append(out, snapshotNode{
					ID: n.ID, Kind: string(n.Kind), Label: n.Label, Value: n.Value, Disabled: n.Disabled,
					Bounds:  [4]int{n.Bounds.Min.X, n.Bounds.Min.Y, n.Bounds.Dx(), n.Bounds.Dy()},
					Actions: actionsOf(n.Target),
					Scroll:  n.Scroll,
				})
			}
			return map[string]any{
				"window": map[string]any{"title": a.Title(), "width": size.X, "height": size.Y, "theme": string(properties.CurrentMode())},
				"nodes":  out,
			}, nil
		})
	snapshotTool.ReadOnly = true
	b.srv.AddTool(snapshotTool)

	shot := mcp.Func("screenshot", "Captures the app window as a PNG image, or with id only one node's area.",
		func(ctx context.Context, in struct {
			ID      string `json:"id,omitempty" description:"Capture only this node from ui_snapshot (its visible part)"`
			Padding int    `json:"padding,omitempty" description:"Pixels of surroundings to include around the node"`
		}) (any, error) {
			ctx, cancel := ctxFor(ctx)
			defer cancel()
			var (
				data []byte
				serr error
			)
			if b.underlayGL.Load() {
				b.takeCapture()
				b.wantCapture.Store(true)
			}
			if err := b.afterFrame(ctx, a, func(ops *op.Ops, size image.Point) {
				area := image.Rectangle{Max: size}
				if in.ID != "" {
					// The node as laid out in this frame.
					n, ok := b.node(in.ID)
					if !ok {
						serr = fmt.Errorf("no node %q on screen; call ui_snapshot again", in.ID)
						return
					}
					area = n.Bounds.Inset(-max(0, in.Padding)).Intersect(area)
					if area.Empty() {
						serr = fmt.Errorf("node %q is outside the window; scroll it into view with ui_scroll", in.ID)
						return
					}
				}
				img := b.takeCapture()
				if img == nil || img.Rect.Size() != size {
					var err error
					if img, err = b.screenshot(ops, size); err != nil {
						serr = err
						return
					}
				}
				data, serr = encodePNG(img.SubImage(area))
			}); err != nil {
				return nil, err
			}
			if serr != nil {
				return nil, serr
			}
			return mcp.Image(data, "image/png"), nil
		})
	shot.ReadOnly = true
	b.srv.AddTool(shot)

	stateList := mcp.Func("state_list", "Lists the app's named states (created with giode.NamedState) and their current JSON values.",
		func(ctx context.Context, in struct{}) (any, error) {
			out := map[string]json.RawMessage{}
			for _, e := range state.Entries() {
				v, err := e.JSON()
				if err != nil {
					v, _ = json.Marshal("<" + err.Error() + ">")
				}
				out[e.Name()] = v
			}
			return out, nil
		})
	stateList.ReadOnly = true
	b.srv.AddTool(stateList)

	frames := mcp.Func("frame_stats", "Reports how long the app's recent frames (up to 240) took: build (running the view and "+
		"laying it out), render (handing the frame to the GPU and presenting it, which waits when the GPU is behind) "+
		"and the interval between frames while they follow each other, as average, 95th percentile and max in "+
		"milliseconds, plus the frame rate. Reset, do the interaction (scroll, stream a reply), then read.",
		func(ctx context.Context, in struct {
			Reset bool `json:"reset,omitempty" description:"Discard the frames measured so far, after reporting them"`
		}) (any, error) {
			st := a.FrameStats()
			if in.Reset {
				a.ResetFrameStats()
			}
			t := func(ft FrameTiming) map[string]float64 {
				m := func(d time.Duration) float64 { return float64(d.Microseconds()) / 1000 }
				return map[string]float64{"avg_ms": m(ft.Avg), "p95_ms": m(ft.P95), "max_ms": m(ft.Max)}
			}
			return map[string]any{
				"frames":   st.Frames,
				"build":    t(st.Build),
				"render":   t(st.Render),
				"interval": t(st.Interval),
				"fps":      math.Round(st.FPS*10) / 10,
				"note":     "Build includes recording the semantic tree for MCP; render can include waiting for the display refresh.",
			}, nil
		})
	frames.ReadOnly = true
	b.srv.AddTool(frames)

	if b.opts.ReadOnly {
		return
	}

	b.registerInputTools(a)
	b.registerFileDragTool(a)

	b.srv.AddTool(mcp.Func("ui_scroll", "Scrolls a scroll area by an amount, or until a node is fully visible. "+
		"Without id it scrolls the main scroll area (the largest one not inside another). "+
		"With until, it scrolls toward the node - paging through lists whose rows aren't laid out yet - and reports whether it's visible.",
		func(ctx context.Context, in struct {
			ID    string `json:"id,omitempty" description:"A scroll node from ui_snapshot, or any node inside one (scrolls its scroll area)"`
			DX    int    `json:"dx,omitempty" description:"Pixels to scroll right (negative: left)"`
			DY    int    `json:"dy,omitempty" description:"Pixels to scroll down (negative: up); with until, the direction to page in when the node isn't found (default down)"`
			Until string `json:"until,omitempty" description:"A node id, or a node's label or value (an exact match first, otherwise text it contains), to bring into view"`
		}) (any, error) {
			ctx, cancel := context.WithTimeout(ctx, 4*b.opts.Timeout)
			defer cancel()
			return b.scroll(ctx, a, in.ID, in.DX, in.DY, in.Until)
		}))

	b.srv.AddTool(mcp.Func("ui_click", "Clicks a node from ui_snapshot: a button, tab, menu item, or toggles a checkbox.",
		func(ctx context.Context, in struct {
			ID string `json:"id" description:"Node id from ui_snapshot"`
		}) (any, error) {
			n, ok := b.node(in.ID)
			if !ok {
				return nil, fmt.Errorf("no node %q in the last snapshot; call ui_snapshot again", in.ID)
			}
			var act func()
			switch t := n.Target.(type) {
			case clicker:
				act = t.Click
			case toggler:
				act = t.Toggle
			default:
				return nil, fmt.Errorf("node %q (%s) can't be clicked", in.ID, n.Kind)
			}
			ctx, cancel := ctxFor(ctx)
			defer cancel()
			if err := a.RunOnUI(ctx, act); err != nil {
				return nil, err
			}
			return "clicked " + in.ID, nil
		}))

	b.srv.AddTool(mcp.Func("ui_type", "Replaces the text of an input node from ui_snapshot, optionally submitting it as Enter would.",
		func(ctx context.Context, in struct {
			ID     string `json:"id" description:"Input node id from ui_snapshot"`
			Text   string `json:"text" description:"The new content"`
			Submit bool   `json:"submit,omitempty" description:"Submit after typing"`
		}) (any, error) {
			n, ok := b.node(in.ID)
			if !ok {
				return nil, fmt.Errorf("no node %q in the last snapshot; call ui_snapshot again", in.ID)
			}
			var set func(string)
			switch t := n.Target.(type) {
			case textChanger:
				set = t.ChangeText
			case textSetter:
				set = t.SetText
			default:
				return nil, fmt.Errorf("node %q (%s) doesn't accept text", in.ID, n.Kind)
			}
			ctx, cancel := ctxFor(ctx)
			defer cancel()
			err := a.RunOnUI(ctx, func() {
				set(in.Text)
				if s, ok := n.Target.(submitter); ok && in.Submit {
					s.SubmitText()
				}
			})
			if err != nil {
				return nil, err
			}
			return "typed into " + in.ID, nil
		}))

	b.srv.AddTool(mcp.Func("ui_set_value", "Sets the value of a checkbox (true/false), slider (0 to 1) or select (option text or index) node from ui_snapshot.",
		func(ctx context.Context, in struct {
			ID    string `json:"id" description:"Node id from ui_snapshot"`
			Value string `json:"value" description:"true/false for checkboxes, 0-1 for sliders, the option (or its index) for selects"`
		}) (any, error) {
			n, ok := b.node(in.ID)
			if !ok {
				return nil, fmt.Errorf("no node %q in the last snapshot; call ui_snapshot again", in.ID)
			}
			var act func()
			switch t := n.Target.(type) {
			case toggler:
				want, err := strconv.ParseBool(in.Value)
				if err != nil {
					return nil, fmt.Errorf("checkbox value must be true or false")
				}
				act = func() {
					if t.Checked() != want {
						t.Toggle()
					}
				}
			case valueChanger:
				v, err := strconv.ParseFloat(in.Value, 32)
				if err != nil {
					return nil, fmt.Errorf("slider value must be a number between 0 and 1")
				}
				act = func() { t.Change(float32(v)) }
			case optionChooser:
				idx := -1
				for i, o := range t.Options() {
					if strings.EqualFold(o, in.Value) {
						idx = i
					}
				}
				if idx < 0 {
					if i, err := strconv.Atoi(in.Value); err == nil {
						idx = i
					}
				}
				if idx < 0 || idx >= len(t.Options()) {
					return nil, fmt.Errorf("unknown option %q; options: %s", in.Value, strings.Join(t.Options(), ", "))
				}
				act = func() { t.Choose(idx) }
			default:
				return nil, fmt.Errorf("node %q (%s) has no settable value", in.ID, n.Kind)
			}
			ctx, cancel := ctxFor(ctx)
			defer cancel()
			if err := a.RunOnUI(ctx, act); err != nil {
				return nil, err
			}
			return "set " + in.ID, nil
		}))

	b.srv.AddTool(mcp.Func("state_set", "Sets a named state to a JSON value of its type.",
		func(ctx context.Context, in struct {
			Name  string          `json:"name" description:"State name from state_list"`
			Value json.RawMessage `json:"value" description:"The new value, as JSON"`
		}) (any, error) {
			e, ok := state.Lookup(in.Name)
			if !ok {
				return nil, fmt.Errorf("no state named %q", in.Name)
			}
			if err := e.SetJSON(in.Value); err != nil {
				return nil, err
			}
			return "set " + in.Name, nil
		}))

	b.srv.AddTool(mcp.Func("set_theme", "Switches the app's color mode.",
		func(ctx context.Context, in struct {
			Mode string `json:"mode" enum:"light,dark"`
		}) (any, error) {
			switch Mode(in.Mode) {
			case ModeLight, ModeDark:
				properties.SetMode(Mode(in.Mode))
				return "theme set to " + in.Mode, nil
			}
			return nil, fmt.Errorf("mode must be light or dark")
		}))

	b.srv.AddTool(mcp.Func("window", "Performs a window action. resize sets the window size in dp, e.g. 400x840 to test a phone layout on a desktop.",
		func(ctx context.Context, in struct {
			Action string `json:"action" enum:"minimize,maximize,unmaximize,fullscreen,windowed,center,raise,resize"`
			Width  int    `json:"width,omitempty" description:"New width in dp (resize only)"`
			Height int    `json:"height,omitempty" description:"New height in dp (resize only)"`
		}) (any, error) {
			switch in.Action {
			case "resize":
				if in.Width <= 0 || in.Height <= 0 {
					return nil, fmt.Errorf("resize needs a positive width and height")
				}
				a.SetSize(in.Width, in.Height)
				return fmt.Sprintf("window resized to %dx%d dp", in.Width, in.Height), nil
			case "minimize":
				a.Minimize()
			case "maximize":
				a.Maximize()
			case "unmaximize":
				a.Unmaximize()
			case "fullscreen":
				a.SetFullscreen(true)
			case "windowed":
				a.SetFullscreen(false)
			case "center":
				a.Center()
			case "raise":
				a.Raise()
			default:
				return nil, fmt.Errorf("unknown action %q", in.Action)
			}
			return "ok", nil
		}))
}
