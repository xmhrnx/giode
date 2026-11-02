// Package tooltip provides a stateful tooltip, shown on hover or
// long-press, that wraps any element as its trigger.
package tooltip

import (
	"image"
	"time"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

var (
	defaultBackground = "#0f172a"
	defaultText       = "#f8fafc"
)

// Touch timings: how long a finger must rest on the trigger to show
// the tooltip, and how long it stays after the finger lifts.
const (
	longPress = 500 * time.Millisecond
	linger    = 1500 * time.Millisecond
)

// Tooltip is a stateful tooltip wrapping trigger: hovering trigger
// with a mouse, or long-pressing it on a touch screen, shows content
// in a small panel anchored just below it, rendered above everything
// else via elements.RegisterPortal. After a long-press the tooltip
// stays for a moment once the finger lifts. Create it once with New
// and lay it out every frame in trigger's place.
type Tooltip struct {
	pass    func(gtx layout.Context) bool
	trigger elements.Element
	content string
	st      styles.Styles
	// placement is the side of the trigger the panel goes.
	placement elements.Placement

	// hovered is set while a mouse is over the trigger.
	hovered bool
	// touchAt is when a finger went down on the trigger; zero when
	// none is down.
	touchAt time.Time
	// longPressed is set once that finger rested for longPress.
	longPressed bool
	// shownUntil keeps the tooltip visible after a long-press ends.
	shownUntil time.Time
}

// New wraps trigger with a tooltip showing content on hover or
// long-press. The styles argument is optional: Background and Color
// style the panel, FontSize its text.
func New(trigger elements.Element, content string, st ...styles.Styles) *Tooltip {
	t := &Tooltip{trigger: trigger, content: content}
	if len(st) > 0 {
		t.st = st[0]
	}
	return t
}

// Placement chooses the side of the trigger the tooltip shows on:
// elements.PlaceAuto (the default) shows it below, or above when it
// wouldn't fit below; PlaceBelow and PlaceAbove keep it on one side.
func (t *Tooltip) Placement(p elements.Placement) *Tooltip {
	t.placement = p
	return t
}

// SetContent replaces the tooltip text.
func (t *Tooltip) SetContent(content string) {
	t.content = content
}

// Hovered reports whether a mouse was over the trigger on the last
// frame.
func (t *Tooltip) Hovered() bool {
	return t.hovered
}

// Visible reports whether the tooltip was showing on the last frame,
// from hover or a long-press.
func (t *Tooltip) Visible() bool {
	return t.hovered || t.longPressed || !t.shownUntil.IsZero()
}

// update processes the pointer events queued on the trigger's area.
func (t *Tooltip) update(gtx layout.Context) {
	for {
		ev, ok := gtx.Event(pointer.Filter{
			Target: t,
			Kinds:  pointer.Enter | pointer.Leave | pointer.Press | pointer.Release | pointer.Cancel,
		})
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		touch := e.Source == pointer.Touch
		switch e.Kind {
		case pointer.Enter:
			if !touch {
				t.hovered = true
			}
		case pointer.Leave:
			if !touch {
				t.hovered = false
			} else {
				// The finger slid off before (or after) the long-press.
				t.touchAt, t.longPressed = time.Time{}, false
			}
		case pointer.Cancel:
			// A scroll took the pointer, or the window lost it.
			t.hovered, t.touchAt, t.longPressed = false, time.Time{}, false
		case pointer.Press:
			if touch {
				t.touchAt = gtx.Now
				t.shownUntil = time.Time{}
				kit.RequestFrameAt(gtx, gtx.Now.Add(longPress))
			}
		case pointer.Release:
			if touch {
				if t.longPressed {
					t.shownUntil = gtx.Now.Add(linger)
				}
				t.touchAt, t.longPressed = time.Time{}, false
			}
		}
	}
	if !t.touchAt.IsZero() && !t.longPressed {
		if gtx.Now.Sub(t.touchAt) >= longPress {
			t.longPressed = true
		} else {
			kit.RequestFrameAt(gtx, t.touchAt.Add(longPress))
		}
	}
	if !t.shownUntil.IsZero() {
		if gtx.Now.Before(t.shownUntil) {
			kit.RequestFrameAt(gtx, t.shownUntil)
		} else {
			t.shownUntil = time.Time{}
		}
	}
}

// Layout lays out the trigger and, while hovered or long-pressed,
// queues the tooltip panel as a portal beside it (see Placement).
func (t *Tooltip) Layout(gtx layout.Context) layout.Dimensions {
	t.update(gtx)
	dims := t.trigger.Layout(gtx)

	// A pass-through area over the trigger sees the pointer without
	// taking clicks from it.
	ps := pointer.PassOp{}.Push(gtx.Ops)
	cl := clip.Rect{Max: dims.Size}.Push(gtx.Ops)
	event.Op(gtx.Ops, t)
	cl.Pop()
	ps.Pop()

	if t.Visible() {
		at := elements.AbsolutePosition()
		trigger := image.Rectangle{Min: at, Max: at.Add(dims.Size)}
		elements.RegisterPortalAround(gtx, trigger, gtx.Dp(6), t.placement, func() elements.Element {
			return elements.Raw(t.panel)
		})
	}
	if t.pass == nil {
		t.pass = func(gtx layout.Context) bool {
			was := t.Visible()
			t.update(gtx)
			return t.Visible() != was
		}
	}
	if !kit.HandleInput(t.pass) && (t.hovered || !t.touchAt.IsZero()) {
		kit.MarkLive()
	}
	return dims
}

// panel renders the tooltip content, hugging it.
func (t *Tooltip) panel(gtx layout.Context) layout.Dimensions {
	st := styles.Merge(styles.Styles{
		Background:   defaultBackground,
		Color:        defaultText,
		Padding:      properties.SymmetricInset(10, 6),
		BorderRadius: 6,
		FontSize:     13,
	}, t.st)
	gtx.Constraints.Min = image.Point{}
	return elements.Box(st, elements.Text(t.content, styles.Styles{
		Color: st.Color, FontSize: st.FontSize,
	})).Layout(gtx)
}
