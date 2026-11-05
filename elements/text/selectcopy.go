package text

import (
	"image"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/widget"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements/box"
	"github.com/xmhrnx/giode/elements/internal/base"
	"github.com/xmhrnx/giode/elements/raw"
	"github.com/xmhrnx/giode/i18n"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// copyAction is the Copy button shown above selected text on touch
// screens and after a right-click.
type copyAction struct {
	btn     *box.Pressable
	dismiss widget.Clickable
	now     bool
}

// layoutCopy registers the press area over the text and, while it's
// all selected, the Copy action above it. text is what Copy copies.
func (s *selectState) layoutCopy(gtx layout.Context, size image.Point, text string) {
	base.TrackPosition(&s.origin)
	s.copyText = text
	// Pass-through, so a scroll under the text keeps scrolling and the
	// text's own handlers keep working.
	ps := pointer.PassOp{}.Push(gtx.Ops)
	cl := clip.Rect{Max: size}.Push(gtx.Ops)
	event.Op(gtx.Ops, &s.tap)
	cl.Pop()
	ps.Pop()
	if !s.all {
		return
	}
	s.copy.show(image.Rectangle{Min: s.origin, Max: s.origin.Add(size)}, s.copyText, func() { s.all = false })
}

func newCopyAction() *copyAction {
	c := &copyAction{}
	c.btn = box.NewPressable(func(p box.PressState) base.Element {
		bg := "#1f2937"
		if p.Pressed || p.Hovered {
			bg = "#374151"
		}
		return box.Box(styles.Styles{Background: bg, BorderRadius: 8, Padding: properties.SymmetricInset(14, 8)},
			Text(i18n.T("giode.copy"), styles.Styles{Color: "#f9fafb", FontSize: 14}))
	}).Label(i18n.T("giode.copy")).OnClick(func(box.ClickEvent) { c.now = true })
	return c
}

// show registers an overlay with the Copy button above (or below) at,
// the selection's bounds in window coordinates, and a tap-anywhere-else
// area. A copy writes text to the clipboard; either ends the selection
// by calling done.
func (c *copyAction) show(at image.Rectangle, text string, done func()) {
	base.RegisterOverlay(func() base.Element {
		return raw.Raw(func(gtx layout.Context) layout.Dimensions {
			win := gtx.Constraints.Max
			if c.now {
				c.now = false
				done()
				kit.WriteClipboard(gtx, text)
				kit.RequestFrame(gtx)
				return layout.Dimensions{}
			}
			if c.dismiss.Clicked(gtx) {
				done()
				kit.RequestFrame(gtx)
				return layout.Dimensions{}
			}
			// A tap anywhere else ends the selection.
			c.dismiss.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Dimensions{Size: win}
			})
			track := base.TrackChild()
			macro := op.Record(gtx.Ops)
			cgtx := gtx
			cgtx.Constraints.Min = image.Point{}
			dims := c.btn.Layout(cgtx)
			call := macro.Stop()
			track.End()
			pos := image.Pt(at.Min.X, at.Min.Y-dims.Size.Y-gtx.Dp(8))
			if pos.Y < gtx.Dp(8) {
				pos.Y = at.Max.Y + gtx.Dp(8)
			}
			pos.X = max(gtx.Dp(8), min(pos.X, win.X-dims.Size.X-gtx.Dp(8)))
			track.Place(pos)
			defer op.Offset(pos).Push(gtx.Ops).Pop()
			call.Add(gtx.Ops)
			return layout.Dimensions{Size: win}
		})
	})
}
