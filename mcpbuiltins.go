package giode

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"math"
	"strconv"
	"strings"
	"time"

	"gioui.org/op"
	"github.com/xmhrnx/giode/mcp"
	"github.com/xmhrnx/giode/state"
	"github.com/xmhrnx/giode/styles/properties"
)

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
