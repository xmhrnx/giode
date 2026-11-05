// Package tooltip provides a stateful tooltip, shown on hover or
// long-press, that wraps any element as its trigger.

package tooltip

import (
	"image"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

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
