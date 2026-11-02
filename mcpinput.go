package giode

import (
	"context"
	"errors"
	"fmt"
	"image"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/op"

	"github.com/mehran9675/giode/mcp"
)

// errInjectUnpatched is what ui_pointer and ui_key return in apps not
// built against Gio patches that can deliver input.
var errInjectUnpatched = errors.New("ui_pointer and ui_key need giode's Gio patches: build and run the app with an up to date giode CLI (giode upgrade, giode run)")

// pointerState is the simulated mouse: where it is and what's held.
type pointerState struct {
	mu      sync.Mutex
	pos     f32.Point
	buttons pointer.Buttons
}

var inputEpoch = time.Now()

// registerInputTools adds ui_pointer and ui_key, which deliver real
// pointer and key events where ui_click and ui_type call components.
func (b *mcpBridge) registerInputTools(a *App) {
	var ptr pointerState
	b.srv.AddTool(mcp.Func("ui_pointer", "Sends mouse events to the window as a user's mouse would: move, press, release, click, "+
		"double_click or scroll, at a node from ui_snapshot (its center) or at window coordinates in pixels. "+
		"Press, move and release drag; events go to whatever is under the pointer, video and custom widgets included.",
		func(ctx context.Context, in struct {
			Action    string   `json:"action" description:"move, press, release, click, double_click or scroll"`
			ID        string   `json:"id,omitempty" description:"Node id from ui_snapshot to point at (its center)"`
			X         *float32 `json:"x,omitempty" description:"Window x in pixels, when there's no id (default: where the pointer is)"`
			Y         *float32 `json:"y,omitempty" description:"Window y in pixels"`
			Button    string   `json:"button,omitempty" description:"primary (default), secondary or tertiary"`
			Modifiers []string `json:"modifiers,omitempty" description:"Held keys: ctrl, shift, alt, super, cmd, shortcut (cmd on macOS, ctrl elsewhere)"`
			DX        float32  `json:"dx,omitempty" description:"Scroll: pixels right (negative: left)"`
			DY        float32  `json:"dy,omitempty" description:"Scroll: pixels down (negative: up)"`
		}) (any, error) {
			mods, err := parseModifiers(in.Modifiers)
			if err != nil {
				return nil, err
			}
			btn, err := parseButton(in.Button)
			if err != nil {
				return nil, err
			}
			ptr.mu.Lock()
			defer ptr.mu.Unlock()
			pos := ptr.pos
			switch {
			case in.ID != "":
				n, ok := b.node(in.ID)
				if !ok {
					return nil, fmt.Errorf("no node %q in the last snapshot; call ui_snapshot again", in.ID)
				}
				c := n.Bounds.Min.Add(n.Bounds.Max).Div(2)
				pos = f32.Pt(float32(c.X), float32(c.Y))
			case in.X != nil || in.Y != nil:
				if in.X != nil {
					pos.X = *in.X
				}
				if in.Y != nil {
					pos.Y = *in.Y
				}
			}
			evs, buttons, err := pointerEvents(in.Action, ptr.pos, pos, ptr.buttons, btn, mods, in.DX, in.DY)
			if err != nil {
				return nil, err
			}
			ptr.buttons = buttons
			ptr.pos = pos
			if err := b.inject(ctx, a, evs...); err != nil {
				return nil, err
			}
			res := map[string]any{"action": in.Action, "position": [2]int{int(pos.X), int(pos.Y)}, "buttons_held": ptr.buttons.String()}
			if in.Action != "move" && in.Action != "scroll" {
				res["button"] = btn.String()
			}
			return res, nil
		}))

	b.srv.AddTool(mcp.Func("ui_key", "Presses and releases a key with modifiers, delivered to the focused element as the keyboard would: "+
		"shortcuts (ctrl+s), navigation (tab, arrows, page_down), enter, escape. Type text into inputs with ui_type.",
		func(ctx context.Context, in struct {
			Key       string   `json:"key" description:"A letter or digit, a symbol, enter, return, tab, escape, space, backspace, delete, up, down, left, right, home, end, page_up, page_down, f1-f12; or a combination like ctrl+shift+z"`
			Modifiers []string `json:"modifiers,omitempty" description:"Held keys: ctrl, shift, alt, super, cmd, shortcut (cmd on macOS, ctrl elsewhere)"`
			State     string   `json:"state,omitempty" description:"press_release (default), press or release"`
		}) (any, error) {
			parts := strings.Split(in.Key, "+")
			if len(in.Key) > 1 && strings.HasSuffix(in.Key, "++") {
				// ctrl++: the plus key.
				parts = append(strings.Split(strings.TrimSuffix(in.Key, "++"), "+"), "+")
			}
			name, err := parseKeyName(parts[len(parts)-1])
			if err != nil {
				return nil, err
			}
			mods, err := parseModifiers(append(parts[:len(parts)-1:len(parts)-1], in.Modifiers...))
			if err != nil {
				return nil, err
			}
			var evs []event.Event
			switch in.State {
			case "", "press_release":
				evs = []event.Event{key.Event{Name: name, Modifiers: mods, State: key.Press}, key.Event{Name: name, Modifiers: mods, State: key.Release}}
			case "press":
				evs = []event.Event{key.Event{Name: name, Modifiers: mods, State: key.Press}}
			case "release":
				evs = []event.Event{key.Event{Name: name, Modifiers: mods, State: key.Release}}
			default:
				return nil, fmt.Errorf("unknown state %q: use press_release, press or release", in.State)
			}
			if err := b.inject(ctx, a, evs...); err != nil {
				return nil, err
			}
			return fmt.Sprintf("sent %s%s", modPrefix(mods), name), nil
		}))
}

// registerFileDragTool adds ui_file_drag, which drags files from the
// file manager over the window: the events a real drag delivers, to
// OnFileDrag, App.FileDrag and OnFileDrop.
func (b *mcpBridge) registerFileDragTool(a *App) {
	var mu sync.Mutex
	var pos f32.Point
	b.srv.AddTool(mcp.Func("ui_file_drag", "Drags files over the window as from the file manager: enter, move, leave, or drop "+
		"with their paths, at a node from ui_snapshot (its center) or at window coordinates in pixels. "+
		"Tests drop targets and their highlight while files are over them; drop alone does enter, move and drop.",
		func(ctx context.Context, in struct {
			Action string   `json:"action" description:"enter, move, leave or drop"`
			ID     string   `json:"id,omitempty" description:"Node id from ui_snapshot to drag over (its center)"`
			X      *float32 `json:"x,omitempty" description:"Window x in pixels, when there's no id (default: where the drag is)"`
			Y      *float32 `json:"y,omitempty" description:"Window y in pixels"`
			Paths  []string `json:"paths,omitempty" description:"Drop: the files' paths"`
		}) (any, error) {
			if !a.fileDragWanted() {
				return nil, errors.New("the app takes no file drops: it sets neither Config.OnFileDrop nor OnFileDrag")
			}
			mu.Lock()
			defer mu.Unlock()
			p := pos
			switch {
			case in.ID != "":
				n, ok := b.node(in.ID)
				if !ok {
					return nil, fmt.Errorf("no node %q in the last snapshot; call ui_snapshot again", in.ID)
				}
				c := n.Bounds.Min.Add(n.Bounds.Max).Div(2)
				p = f32.Pt(float32(c.X), float32(c.Y))
			case in.X != nil || in.Y != nil:
				if in.X != nil {
					p.X = *in.X
				}
				if in.Y != nil {
					p.Y = *in.Y
				}
			}
			_, dragging := a.FileDrag()
			switch in.Action {
			case "enter":
				a.deliverFileDrag(FileDragEnter, p)
			case "move":
				if !dragging {
					a.deliverFileDrag(FileDragEnter, p)
				}
				a.deliverFileDrag(FileDragOver, p)
			case "leave":
				a.deliverFileDrag(FileDragLeave, p)
			case "drop":
				if len(in.Paths) == 0 {
					return nil, errors.New("give the dropped files' paths")
				}
				if !dragging {
					a.deliverFileDrag(FileDragEnter, p)
					a.deliverFileDrag(FileDragOver, p)
				}
				a.deliverFileDrop(in.Paths, p)
			default:
				return nil, fmt.Errorf("unknown action %q: use enter, move, leave or drop", in.Action)
			}
			pos = p
			ctx, cancel := context.WithTimeout(ctx, b.opts.Timeout)
			defer cancel()
			if err := b.afterFrame(ctx, a, func(*op.Ops, image.Point) {}); err != nil {
				return nil, err
			}
			_, dragging = a.FileDrag()
			return map[string]any{"action": in.Action, "position": [2]int{int(p.X), int(p.Y)}, "dragging": dragging}, nil
		}))
}

// inject delivers events and waits for the frame that handles them.
func (b *mcpBridge) inject(ctx context.Context, a *App, evs ...event.Event) error {
	ctx, cancel := context.WithTimeout(ctx, b.opts.Timeout)
	defer cancel()
	for _, e := range evs {
		if err := injectEvent(ctx, a, e); err != nil {
			return err
		}
	}
	return b.afterFrame(ctx, a, func(*op.Ops, image.Point) {})
}

// pointerEvents builds the events for a ui_pointer action at pos, from
// the pointer at prev with held buttons, using button btn: a move when
// the pointer moved, then the presses and releases, each carrying the
// buttons held at that moment. It returns the buttons held afterwards.
func pointerEvents(action string, prev, pos f32.Point, held, btn pointer.Buttons, mods key.Modifiers, dx, dy float32) ([]event.Event, pointer.Buttons, error) {
	ev := func(kind pointer.Kind) pointer.Event {
		return pointer.Event{Kind: kind, Source: pointer.Mouse, Position: pos, Buttons: held,
			Modifiers: mods, Time: time.Since(inputEpoch)}
	}
	var evs []event.Event
	if pos != prev || action == "move" {
		evs = append(evs, ev(pointer.Move))
	}
	press := func() {
		held |= btn
		evs = append(evs, ev(pointer.Press))
	}
	release := func() {
		held &^= btn
		evs = append(evs, ev(pointer.Release))
	}
	switch action {
	case "move":
	case "press":
		press()
	case "release":
		release()
	case "click":
		press()
		release()
	case "double_click":
		press()
		release()
		press()
		release()
	case "scroll":
		e := ev(pointer.Scroll)
		e.Scroll = f32.Pt(dx, dy)
		evs = append(evs, e)
	default:
		return nil, held, fmt.Errorf("unknown action %q: use move, press, release, click, double_click or scroll", action)
	}
	return evs, held, nil
}

func parseButton(s string) (pointer.Buttons, error) {
	switch strings.ToLower(s) {
	case "", "primary", "left":
		return pointer.ButtonPrimary, nil
	case "secondary", "right":
		return pointer.ButtonSecondary, nil
	case "tertiary", "middle":
		return pointer.ButtonTertiary, nil
	}
	return 0, fmt.Errorf("unknown button %q: use primary, secondary or tertiary", s)
}

func parseModifiers(names []string) (key.Modifiers, error) {
	var m key.Modifiers
	for _, n := range names {
		switch strings.ToLower(strings.TrimSpace(n)) {
		case "ctrl", "control":
			m |= key.ModCtrl
		case "shift":
			m |= key.ModShift
		case "alt", "option", "opt":
			m |= key.ModAlt
		case "super", "win", "meta":
			m |= key.ModSuper
		case "cmd", "command":
			m |= key.ModCommand
		case "shortcut", "mod":
			m |= key.ModShortcut
		case "":
		default:
			return 0, fmt.Errorf("unknown modifier %q: use ctrl, shift, alt, super, cmd or shortcut", n)
		}
	}
	return m, nil
}

var keyNames = map[string]key.Name{
	"enter": key.NameReturn, "return": key.NameReturn, "keypad_enter": key.NameEnter,
	"tab": key.NameTab, "escape": key.NameEscape, "esc": key.NameEscape, "space": key.NameSpace,
	"backspace": key.NameDeleteBackward, "delete": key.NameDeleteForward, "del": key.NameDeleteForward,
	"up": key.NameUpArrow, "down": key.NameDownArrow, "left": key.NameLeftArrow, "right": key.NameRightArrow,
	"home": key.NameHome, "end": key.NameEnd, "page_up": key.NamePageUp, "pageup": key.NamePageUp,
	"page_down": key.NamePageDown, "pagedown": key.NamePageDown, "back": key.NameBack,
}

func parseKeyName(s string) (key.Name, error) {
	if n, ok := keyNames[strings.ToLower(s)]; ok {
		return n, nil
	}
	if len(s) >= 2 && (s[0] == 'f' || s[0] == 'F') {
		var n int
		if _, err := fmt.Sscanf(s[1:], "%d", &n); err == nil && n >= 1 && n <= 12 && fmt.Sprint(n) == s[1:] {
			return key.Name(fmt.Sprintf("F%d", n)), nil
		}
	}
	if utf8.RuneCountInString(s) == 1 {
		// Gio names letter keys in upper case.
		return key.Name(strings.ToUpper(s)), nil
	}
	return "", fmt.Errorf("unknown key %q", s)
}

func modPrefix(m key.Modifiers) string {
	if m == 0 {
		return ""
	}
	return m.String() + "+"
}
