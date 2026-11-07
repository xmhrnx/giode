package box

import (
	"image"
	"time"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"

	"github.com/xmhrnx/giode/elements/internal/base"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
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
	// Contextual gives it a context menu (Context).
	base.Contextual

	content func(PressState) base.Element
	st      styles.Styles

	onClick, onDouble, onSecondary func(ClickEvent)
	onLongPress                    func(ClickEvent)
	onDragStart, onDragMove        func(DragEvent)
	onDragEnd                      func(DragEvent)
	label                          string
	cursor                         properties.Cursor
	disabled                       bool
	focusable                      bool
	// selectable keeps the content's text selectable, as in a clickable
	// Box (OnClick): a mouse drag across it selects rather than clicks.
	selectable bool

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

	// pressID is the pointer that pressed; pressWin where it pressed,
	// in the window. dragging is set once that press moved past
	// dragSlop with drag handlers set.
	pressID  pointer.ID
	pressWin f32.Point
	dragging bool
	// size is the last frame's size, for keyboard clicks' position.
	size image.Point
}

// NewPressable returns a Pressable around content. The styles argument
// is optional and styles a box wrapped around the content: padding,
// background, FlexGrow, Width, ... and Cursor (default pointer).
func NewPressable(content func(s PressState) base.Element, st ...styles.Styles) *Pressable {
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

// OnDragStart sets the handler for the start of a drag: a press that
// moves more than a few dp with any button, mouse or finger. Once a
// press becomes a drag, its release is not a click, so a draggable
// tab or card still clicks when it's only tapped. The event's Position
// is where the press was. Setting any drag handler keeps the content's
// text from being selected by a drag. It returns p for chaining.
func (p *Pressable) OnDragStart(fn func(e DragEvent)) *Pressable {
	p.onDragStart = fn
	return p
}

// OnDragMove sets the handler for each move of the pointer during a
// drag, the first one right after OnDragStart. The drag goes on when
// the pointer leaves the Pressable, and once it has started, an
// enclosing Scroll or frameless window doesn't take it. (A finger
// moving along a Scroll's axis scrolls it before it moves far enough
// to start a drag.) It returns p for chaining.
func (p *Pressable) OnDragMove(fn func(e DragEvent)) *Pressable {
	p.onDragMove = fn
	return p
}

// OnDragEnd sets the handler for the end of a drag: the release, or a
// cancellation (DragEvent.Cancelled). It returns p for chaining.
func (p *Pressable) OnDragEnd(fn func(e DragEvent)) *Pressable {
	p.onDragEnd = fn
	return p
}

// draggable reports whether any drag handler is set.
func (p *Pressable) draggable() bool {
	return p.onDragStart != nil || p.onDragMove != nil || p.onDragEnd != nil
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
		p.cancelDrag()
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
func (p *Pressable) Content(content func(s PressState) base.Element) *Pressable {
	p.content = content
	return p
}

// Click runs the click handlers as a programmatic click. Call it on
// the UI goroutine (from a handler, or through App.Post).
func (p *Pressable) Click() {
	if p.disabled {
		return
	}
	p.fire(ClickEvent{Clicks: 1, Source: ClickProgrammatic, WindowPosition: p.below()})
}

// Hovered reports whether the pointer is over it.
func (p *Pressable) Hovered() bool { return p.hovered }

// Pressed reports whether a button is held down on it.
func (p *Pressable) Pressed() bool { return p.pressed }

// Focused reports whether it has keyboard focus.
func (p *Pressable) Focused() bool { return p.focused }

// State returns the current PressState.
func (p *Pressable) State() PressState {
	return PressState{Hovered: p.hovered, Pressed: p.pressed, Focused: p.focused, Disabled: p.disabled, Dragging: p.dragging}
}

// Dragging reports whether a drag is in progress.
func (p *Pressable) Dragging() bool { return p.dragging }

// below is the Pressable's bottom-left corner in the window, where a
// menu opened by a keyboard click goes.
func (p *Pressable) below() image.Point {
	return p.origin.Add(image.Pt(0, p.size.Y))
}

func (p *Pressable) FlexWeight(axis layout.Axis) int  { return base.WeightOf(p.st, axis) }
func (p *Pressable) FlexPercent(axis layout.Axis) int { return base.PercentOf(p.st, axis) }
func (p *Pressable) ZIndex() int                      { return int(p.st.ZIndex) }
