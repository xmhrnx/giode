package elements

import (
	"image"
	"sync/atomic"
	"time"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/semantic"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"

	"github.com/mehran9675/giode/components/kit"
	"github.com/mehran9675/giode/styles"
	"github.com/mehran9675/giode/styles/properties"
)

// PressState is the interaction state a Pressable's content is built
// with, so it can restyle itself.
type PressState struct {
	Hovered  bool
	Pressed  bool
	Focused  bool
	Disabled bool
}

// MouseButton identifies which button produced a click.
type MouseButton int

const (
	MousePrimary MouseButton = iota
	MouseSecondary
	MouseMiddle
)

// ClickSource says how a click was produced.
type ClickSource int

const (
	// ClickPointer is a mouse, pen or touch click.
	ClickPointer ClickSource = iota
	// ClickKeyboard is Enter or Space on a focused Pressable.
	ClickKeyboard
	// ClickProgrammatic is a call to Pressable.Click (or automation).
	ClickProgrammatic
)

// ClickEvent describes a click.
type ClickEvent struct {
	// Clicks is 1 for a single click, 2 for a double click, and so on.
	Clicks int
	// Button is the mouse button; MousePrimary for keyboard and
	// programmatic clicks.
	Button MouseButton
	// Modifiers holds the keys held down (key.ModCtrl, key.ModShift,
	// ...); see Ctrl and Shift.
	Modifiers key.Modifiers
	// Source says how the click was produced.
	Source ClickSource
	// Position is where the pointer was, relative to the Pressable,
	// in pixels; zero for keyboard and programmatic clicks.
	Position image.Point
	// PositionDp is Position in dp, the unit of Styles.
	PositionDp f32.Point
	// WindowPosition is where the pointer was in the window, in pixels:
	// what Menu.OpenAt takes, so a row can open a menu where it was
	// right-clicked.
	WindowPosition image.Point
}

// toDp converts a pixel position to dp.
func toDp(pt f32.Point, gtx layout.Context) f32.Point {
	if s := gtx.Metric.PxPerDp; s > 0 {
		return pt.Div(s)
	}
	return pt
}

// Ctrl reports whether Ctrl (or Command on macOS) was held.
func (e ClickEvent) Ctrl() bool {
	return e.Modifiers.Contain(key.ModShortcut)
}

// Shift reports whether Shift was held.
func (e ClickEvent) Shift() bool {
	return e.Modifiers.Contain(key.ModShift)
}

// multiClickWindow is the longest gap between clicks that still counts
// them as a double (or triple) click, and multiClickSlop how far apart
// they may be, in dp.
const (
	multiClickWindow = 400 * time.Millisecond
	multiClickSlop   = 6
)

// Pressable makes any content clickable. Its content is rebuilt every
// frame with the current PressState, so it can restyle itself:
//
//	card := giode.Pressable(func(s giode.PressState) giode.Element {
//		bg := "theme.surface"
//		if s.Hovered {
//			bg = "theme.border"
//		}
//		return giode.Box(giode.Styles{Background: bg, Padding: giode.UniformInset(12)},
//			giode.Text("Open project"))
//	}).OnClick(func(e giode.ClickEvent) { open() })
//
// The hit area sits beneath the content, like Gio's widget.Clickable,
// so clickable children (an expand arrow, a delete button) keep their
// own clicks. It shows a pointer cursor on hover, can be focused with
// Tab and activated with Enter or Space, stays clickable inside a
// frameless window's drag area, and appears to automation (the MCP
// server) as a button.
//
// Like every stateful component, create it once and reuse it every
// frame - or, for items of a list that changes, look it up by key with
// Keyed.
type Pressable struct {
	content func(PressState) Element
	st      styles.Styles

	onClick, onDouble, onSecondary func(ClickEvent)
	onLongPress                    func(ClickEvent)
	label                          string
	cursor                         properties.Cursor
	disabled                       bool
	focusable                      bool

	hovered  bool
	pressed  bool
	pressBtn MouseButton
	pressPos f32.Point
	// origin is the pressable's window position, exact as of the last
	// frame (TrackPosition).
	origin     image.Point
	focused    bool
	pressedKey key.Name

	lastClick    time.Time
	lastClickPos f32.Point
	clicks       int

	// pressAt is when the current touch press started, for OnLongPress;
	// longFired is set once it fired, so the release isn't a click.
	pressAt  time.Time
	pressSrc pointer.Source
	// fired counts handler calls, so an input pass can report them;
	// passFn is the pass, registered with enclosing caches.
	fired     uint64
	passFn    func(gtx layout.Context) bool
	longFired bool
}

// LongPressDuration is how long a touch must be held still for
// OnLongPress.
const LongPressDuration = 500 * time.Millisecond

// longPressFeedback plays when a long-press fires; the app sets it to
// haptic feedback.
var longPressFeedback atomic.Pointer[func()]

// SetLongPressFeedback sets what plays when OnLongPress fires - the app
// sets it to a haptic tap; component authors don't need to.
func SetLongPressFeedback(fn func()) {
	if fn == nil {
		longPressFeedback.Store(nil)
		return
	}
	longPressFeedback.Store(&fn)
}

// NewPressable returns a Pressable around content. The styles argument
// is optional and styles a box wrapped around the content: padding,
// background, FlexGrow, Width, ... and Cursor (default pointer).
func NewPressable(content func(s PressState) Element, st ...styles.Styles) *Pressable {
	p := &Pressable{content: content, focusable: true}
	if len(st) > 0 {
		p.st = st[0]
	}
	return p
}

// OnClick sets the handler for clicks of any button and count. It
// returns p for chaining.
func (p *Pressable) OnClick(fn func(e ClickEvent)) *Pressable {
	p.onClick = fn
	return p
}

// OnDoubleClick sets a handler for primary-button double clicks, run
// in addition to OnClick. It returns p for chaining.
func (p *Pressable) OnDoubleClick(fn func(e ClickEvent)) *Pressable {
	p.onDouble = fn
	return p
}

// OnSecondaryClick sets a handler for right clicks, run in addition to
// OnClick. It returns p for chaining.
func (p *Pressable) OnSecondaryClick(fn func(e ClickEvent)) *Pressable {
	p.onSecondary = fn
	return p
}

// OnLongPress sets the handler for a touch held on the element for
// LongPressDuration without sliding off it - the touch screen's way to
// open context actions. The release that follows isn't a click. With a
// mouse, use OnSecondaryClick. It returns p for chaining.
func (p *Pressable) OnLongPress(fn func(e ClickEvent)) *Pressable {
	p.onLongPress = fn
	return p
}

// Styles replaces the styles of the box around the content, from the
// next frame. It returns p for chaining.
func (p *Pressable) Styles(st styles.Styles) *Pressable {
	p.st = st
	return p
}

// Label sets the accessible name reported to assistive technology and
// automation. It returns p for chaining.
func (p *Pressable) Label(label string) *Pressable {
	p.label = label
	return p
}

// Disabled turns input off: no events, no focus, default cursor, and
// PressState.Disabled set for the content. It returns p for chaining.
func (p *Pressable) Disabled(v bool) *Pressable {
	p.disabled = v
	if v {
		p.hovered, p.pressed = false, false
	}
	return p
}

// Focusable controls whether Tab can focus it (default true). It
// returns p for chaining.
func (p *Pressable) Focusable(v bool) *Pressable {
	p.focusable = v
	return p
}

// Content replaces the content builder. It returns p for chaining.
func (p *Pressable) Content(content func(s PressState) Element) *Pressable {
	p.content = content
	return p
}

// Click runs the click handlers as a programmatic click. Call it on
// the UI goroutine (from a handler, or through App.Post).
func (p *Pressable) Click() {
	if p.disabled {
		return
	}
	p.fire(ClickEvent{Clicks: 1, Source: ClickProgrammatic})
}

// Hovered reports whether the pointer is over it.
func (p *Pressable) Hovered() bool { return p.hovered }

// Pressed reports whether a button is held down on it.
func (p *Pressable) Pressed() bool { return p.pressed }

// Focused reports whether it has keyboard focus.
func (p *Pressable) Focused() bool { return p.focused }

// State returns the current PressState.
func (p *Pressable) State() PressState {
	return PressState{Hovered: p.hovered, Pressed: p.pressed, Focused: p.focused, Disabled: p.disabled}
}

func (p *Pressable) flexWeight(axis layout.Axis) int { return weightOf(p.st, axis) }
func (p *Pressable) percentOf(axis layout.Axis) int  { return percentOf(p.st, axis) }
func (p *Pressable) zIndex() int                     { return int(p.st.ZIndex) }

func (p *Pressable) fire(e ClickEvent) {
	p.fired++
	if p.onClick != nil {
		p.onClick(e)
	}
	if e.Clicks == 2 && e.Button == MousePrimary && p.onDouble != nil {
		p.onDouble(e)
	}
	if e.Button == MouseSecondary && p.onSecondary != nil {
		p.onSecondary(e)
	}
}

// update processes the events queued since the last frame.
func (p *Pressable) update(gtx layout.Context) (changed bool) {
	if p.disabled {
		return false
	}
	for {
		ev, ok := gtx.Event(pointer.Filter{
			Target: p,
			Kinds:  pointer.Press | pointer.Release | pointer.Enter | pointer.Leave | pointer.Cancel,
		})
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		changed = true
		switch e.Kind {
		case pointer.Enter:
			p.hovered = true
		case pointer.Leave:
			p.hovered = false
		case pointer.Cancel:
			p.hovered, p.pressed = false, false
		case pointer.Press:
			if p.pressed {
				break
			}
			p.pressed = true
			p.pressPos = e.Position
			p.pressSrc = e.Source
			p.pressAt = gtx.Now
			p.longFired = false
			if e.Source == pointer.Touch && p.onLongPress != nil {
				kit.RequestFrameAt(gtx, gtx.Now.Add(LongPressDuration))
			}
			switch {
			case e.Buttons.Contain(pointer.ButtonSecondary):
				p.pressBtn = MouseSecondary
			case e.Buttons.Contain(pointer.ButtonTertiary):
				p.pressBtn = MouseMiddle
			default:
				p.pressBtn = MousePrimary
			}
			if p.focusable && e.Source == pointer.Mouse {
				gtx.Execute(key.FocusCmd{Tag: p})
			}
		case pointer.Release:
			if !p.pressed {
				break
			}
			p.pressed = false
			// Released outside: not a click. Gio keeps a pressed
			// touch's Enter/Leave up to date as the finger slides, and
			// delivers the release before the touch's final Leave, so
			// this holds for touch as well as the mouse.
			if !p.hovered || p.longFired {
				break
			}
			slop := float32(gtx.Dp(multiClickSlop))
			d := e.Position.Sub(p.lastClickPos)
			if gtx.Now.Sub(p.lastClick) <= multiClickWindow && d.X*d.X+d.Y*d.Y <= slop*slop {
				p.clicks++
			} else {
				p.clicks = 1
			}
			p.lastClick, p.lastClickPos = gtx.Now, e.Position
			p.fire(ClickEvent{
				Clicks: p.clicks, Button: p.pressBtn, Modifiers: e.Modifiers,
				Source: ClickPointer, Position: e.Position.Round(), PositionDp: toDp(e.Position, gtx),
				WindowPosition: p.origin.Add(e.Position.Round()),
			})
		}
	}
	if p.pressed && p.pressSrc == pointer.Touch && p.onLongPress != nil && !p.longFired {
		switch {
		case !p.hovered:
			// Slid off: neither a long-press nor a click.
		case gtx.Now.Sub(p.pressAt) >= LongPressDuration:
			p.longFired = true
			changed = true
			if fn := longPressFeedback.Load(); fn != nil {
				(*fn)()
			}
			p.fired++
			p.onLongPress(ClickEvent{Clicks: 1, Source: ClickPointer, Position: p.pressPos.Round(), PositionDp: toDp(p.pressPos, gtx), WindowPosition: p.origin.Add(p.pressPos.Round())})
		default:
			kit.RequestFrameAt(gtx, p.pressAt.Add(LongPressDuration))
		}
	}
	if p.focusable {
		for {
			ev, ok := gtx.Event(
				key.FocusFilter{Target: p},
				key.Filter{Focus: p, Name: key.NameReturn},
				key.Filter{Focus: p, Name: key.NameEnter},
				key.Filter{Focus: p, Name: key.NameSpace},
			)
			if !ok {
				break
			}
			changed = true
			switch e := ev.(type) {
			case key.FocusEvent:
				p.focused = e.Focus
				p.pressedKey = ""
			case key.Event:
				if !gtx.Focused(p) {
					break
				}
				switch e.State {
				case key.Press:
					p.pressedKey = e.Name
				case key.Release:
					if p.pressedKey == e.Name {
						p.pressedKey = ""
						p.fire(ClickEvent{Clicks: 1, Modifiers: e.Modifiers, Source: ClickKeyboard})
					}
				}
			}
		}
	} else {
		p.focused = false
	}
	if changed {
		kit.RequestFrame(gtx)
	}
	return changed
}

// pass is the Pressable's input pass (see kit.HandleInput): it handles
// clicks, hover and focus while an enclosing cache replays it, and
// reports a change in the state its content is built from, or a fired
// handler.
func (p *Pressable) pass(gtx layout.Context) bool {
	before, fired := p.State(), p.fired
	if !p.update(gtx) {
		return false
	}
	return p.State() != before || p.fired != fired
}

// Layout lays out the content over the hit area.
func (p *Pressable) Layout(gtx layout.Context) layout.Dimensions {
	p.update(gtx)
	if p.st.Display == properties.DisplayNone {
		return layout.Dimensions{}
	}
	state := p.State()
	var content Element = Box(styles.Styles{})
	if p.content != nil {
		if c := p.content(state); c != nil {
			content = c
		}
	}
	if p.st != (styles.Styles{}) {
		content = Box(p.st, content)
	}

	TrackPosition(&p.origin)
	macro := op.Record(gtx.Ops)
	dims := kit.Unselectable(content.Layout)(gtx)
	call := macro.Stop()

	// Hit area first, content on top: interactive children win.
	if !p.disabled {
		cl := clip.Rect{Max: dims.Size}.Push(gtx.Ops)
		semantic.Button.Add(gtx.Ops)
		if p.label != "" {
			semantic.LabelOp(p.label).Add(gtx.Ops)
		}
		event.Op(gtx.Ops, p)
		if p.hovered {
			cursor := p.st.Cursor
			if cursor == "" {
				cursor = p.cursor
			}
			if cursor == "" {
				cursor = properties.CursorPointer
			}
			cursor.CursorFor().Add(gtx.Ops)
		}
		kit.BlockDragHere(gtx)
		cl.Pop()
	} else {
		cl := clip.Rect{Max: dims.Size}.Push(gtx.Ops)
		semantic.EnabledOp(false).Add(gtx.Ops)
		cl.Pop()
	}
	call.Add(gtx.Ops)

	// Enclosing caches deliver input through the pass; without one
	// (or in an opaque subtree), stay live while interacting.
	if p.passFn == nil {
		p.passFn = p.pass
	}
	if !kit.HandleInput(p.passFn) && (p.hovered || p.pressed || p.focused) {
		kit.MarkLive()
	}
	if p.onClick != nil || p.onDouble != nil || p.onSecondary != nil || p.onLongPress != nil {
		// Only something that reacts to clicks is a button to
		// automation; a Pressable used just for hover state isn't.
		RecordSemantic(gtx, SemanticNode{Kind: SemanticButton, Label: p.label, Disabled: p.disabled, Target: p}, dims.Size)
	}
	return dims
}

// --- Box sugar -----------------------------------------------------------------

// OnClick makes the box clickable, as if wrapped in a Pressable: the
// pointer cursor shows on hover, Enter/Space activate it when
// focused, and clickable children keep their own clicks. Like OnHover,
// it needs a box created once and reused every frame (or looked up
// with Keyed), since press state lives in the element.
func (b *BoxElement) OnClick(fn func(e ClickEvent)) *BoxElement {
	if b.press == nil {
		b.press = NewPressable(nil)
	}
	b.press.OnClick(fn)
	return b
}

// Pressable returns the Pressable behind OnClick, to set a label,
// double-click handlers or disable it; nil before OnClick.
func (b *BoxElement) Pressable() *Pressable {
	return b.press
}
