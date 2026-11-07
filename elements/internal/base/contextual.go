package base

import (
	"image"
	"time"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"

	"github.com/xmhrnx/giode/components/kit"
)

// ContextMenu is a menu an element opens at the pointer when it's
// right-clicked or long-pressed: a *giode.MenuElement (Menu).
type ContextMenu interface {
	Element
	// OpenAtWindow opens the menu at p, in window pixels.
	OpenAtWindow(p image.Point)
	// Opened reports whether the menu shows.
	Opened() bool
}

// Contextual gives an element a context menu: every element and
// component embeds it, so each has a Context method.
//
//	row := giode.Box(st, giode.Text(name))
//	row.Context(rowMenu)
//
// A right-click anywhere on the element - or a long-press on a touch
// screen - opens the menu where the pointer is, however small the
// element and wherever it sits. With elements inside each other, the
// innermost with a menu opens its own.
type Contextual struct {
	menu ContextMenu
}

// contextState is the presses over one of the elements a menu is
// anchored to. Views build elements anew every frame, so it isn't kept
// in the element: anchorState finds it by the menu and the element's
// turn among the menu's elements in the frame.
type contextState struct {
	menu ContextMenu
	// tag receives the presses over the element.
	tag int
	// origin is the element's window position, exact from the last
	// frame (TrackPosition).
	origin image.Point
	pass   func(gtx layout.Context) bool

	// A touch held on the element, for a long-press.
	holding bool
	holdAt  time.Time
	holdPos f32.Point
	holdID  pointer.ID
}

// Context anchors m to the element: a right-click on it, or a
// long-press, opens m at the pointer. Context(nil) removes it.
func (c *Contextual) Context(m ContextMenu) {
	c.menu = m
}

// ContextMenu returns the menu Context anchored, or nil.
func (c *Contextual) ContextMenu() ContextMenu {
	return c.menu
}

// AnchorContext makes the element's area, size from the current
// offset, open its context menu. Every element calls it at the end of
// its Layout, outside any cache of its own; it does nothing without a
// menu.
func (c *Contextual) AnchorContext(gtx layout.Context, size image.Point) {
	m := c.menu
	if m == nil {
		return
	}
	st := anchorState(m)
	// A cache replaying the element takes its turn for it, so the
	// elements laid out after it keep theirs.
	RetainKeyed(func() { anchorState(m) })
	st.anchor(gtx, size)
}

// menuAnchors are the states of a menu's elements, in the order they
// were laid out; used counts those laid out this frame.
type menuAnchors struct {
	states []*contextState
	used   int
	frame  uint64
}

var anchors = map[ContextMenu]*menuAnchors{}

// anchorState returns the state of the next element m is anchored to
// this frame: the same element, frame after frame, as long as the view
// lays them out in the same order.
func anchorState(m ContextMenu) *contextState {
	a := anchors[m]
	if a == nil {
		a = &menuAnchors{}
		anchors[m] = a
	}
	if a.frame != frameNumber {
		a.frame, a.used = frameNumber, 0
	}
	if a.used == len(a.states) {
		a.states = append(a.states, &contextState{menu: m})
	}
	st := a.states[a.used]
	a.used++
	return st
}

// sweepAnchors forgets the menus no element was anchored to for a
// while.
func sweepAnchors() {
	for m, a := range anchors {
		if frameNumber-a.frame > 600 {
			delete(anchors, m)
		}
	}
}

func (c *contextState) anchor(gtx layout.Context, size image.Point) {
	c.update(gtx)
	TrackPosition(&c.origin)
	// Over the element, passing presses on to it: its own handlers,
	// right-click ones too, still get them.
	ps := pointer.PassOp{}.Push(gtx.Ops)
	cl := clip.Rect{Max: size}.Push(gtx.Ops)
	event.Op(gtx.Ops, &c.tag)
	cl.Pop()
	ps.Pop()
	if c.pass == nil {
		c.pass = c.update
	}
	// An enclosing cache replaying the element delivers its presses.
	kit.HandleInput(c.pass)
	if c.holding {
		kit.MarkLive()
	}
}

// update handles the presses over the element, opening its menu, and
// reports whether it did.
func (c *contextState) update(gtx layout.Context) bool {
	opened := false
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &c.tag, Kinds: pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel})
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		switch e.Kind {
		case pointer.Press:
			c.holding = false
			if claimed(e) {
				// An element inside this one has its own menu.
				continue
			}
			switch {
			case e.Source == pointer.Touch:
				c.holding, c.holdAt, c.holdPos, c.holdID = true, gtx.Now, e.Position, e.PointerID
				kit.RequestFrameAt(gtx, gtx.Now.Add(contextHold))
			case e.Buttons.Contain(pointer.ButtonSecondary):
				claim(e)
				c.open(e.Position)
				opened = true
			}
		case pointer.Drag:
			if c.holding && e.PointerID == c.holdID {
				d := e.Position.Sub(c.holdPos)
				if slop := float32(gtx.Dp(contextSlop)); d.X*d.X+d.Y*d.Y > slop*slop {
					c.holding = false
				}
			}
		case pointer.Release, pointer.Cancel:
			c.holding = false
		}
	}
	if c.holding {
		if gtx.Now.Sub(c.holdAt) >= contextHold {
			c.holding = false
			hold := pointer.Event{Kind: pointer.Press, Source: pointer.Touch, PointerID: c.holdID, Time: time.Duration(c.holdAt.UnixNano())}
			if !claimed(hold) {
				claim(hold)
				c.open(c.holdPos)
				opened = true
			}
		} else {
			kit.RequestFrameAt(gtx, c.holdAt.Add(contextHold))
		}
	}
	return opened
}

// open opens the menu at p, relative to the element.
func (c *contextState) open(p f32.Point) {
	c.menu.OpenAtWindow(c.origin.Add(p.Round()))
	openMenus = append(openMenus, c.menu)
}

// contextHold is how long a finger rests on an element to open its
// menu, and contextSlop how far, in dp, it may move meanwhile.
const (
	contextHold = 500 * time.Millisecond
	contextSlop = 8
)

// The press an element's menu opened for, so the elements around it
// leave it: elements inside others take their presses first, as
// they're laid out (and their passes run) first.
//
// A claim lasts the frame it's made in: every handler reads a press in
// the frame after it arrives, so a later press with the same time and
// pointer (as synthetic events have) isn't mistaken for it.
var claimedPress struct {
	time  time.Duration
	id    pointer.ID
	frame uint64
	set   bool
}

func claim(e pointer.Event) {
	claimedPress.time, claimedPress.id = e.Time, e.PointerID
	claimedPress.frame, claimedPress.set = frameNumber, true
}

func claimed(e pointer.Event) bool {
	return claimedPress.set && claimedPress.frame == frameNumber &&
		claimedPress.time == e.Time && claimedPress.id == e.PointerID
}

// ContextClaimed reports whether an element opened its context menu
// for e, a press: a window-wide menu (App.SetContextMenu) leaves it.
func ContextClaimed(e pointer.Event) bool {
	return claimed(e)
}

// openMenus are the context menus elements opened, laid out every
// frame while they show (see DrainContextMenus): the element that
// opened one may be replayed from a cache, scrolled away or gone.
var openMenus []ContextMenu

// DrainContextMenus lays out the context menus that show, before the
// portals they draw through. The app (and ui.Frame) calls it once per
// frame, after the view.
func DrainContextMenus(gtx layout.Context) {
	if len(openMenus) == 0 {
		return
	}
	kept := openMenus[:0]
	seen := map[ContextMenu]bool{}
	for _, m := range openMenus {
		if seen[m] || !m.Opened() {
			continue
		}
		seen[m] = true
		kept = append(kept, m)
		if l, ok := m.(interface{ LastLaidOut() uint64 }); ok && l.LastLaidOut() == frameNumber {
			// The view laid it out already.
			continue
		}
		mgtx := gtx
		mgtx.Constraints.Min = image.Point{}
		m.Layout(mgtx)
	}
	clear(openMenus[len(kept):])
	openMenus = kept
}
