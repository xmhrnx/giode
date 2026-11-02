// Package badge provides badges: small counters or dots, alone or
// pinned to the corner of another element.
package badge

import (
	"github.com/xmhrnx/giode/components/kit"
	"image"

	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Badge is a small pill holding a short text, such as an unread count,
// or a dot when the text is empty. It's stateless: build it in the
// view function every frame.
type Badge struct {
	text   string
	st     styles.Styles
	target elements.Element
}

// New returns a badge. The styles argument is optional: Background
// (default theme.danger), Color (default white), FontSize (default 11).
func New(text string, st ...styles.Styles) *Badge {
	b := &Badge{text: text}
	if len(st) > 0 {
		b.st = st[0]
	}
	return b
}

// On pins the badge to the top-right corner of el, overlapping it, as
// on an icon or avatar. It returns b for chaining.
func (b *Badge) On(el elements.Element) *Badge {
	b.target = el
	return b
}

func (b *Badge) pill() elements.Element {
	st := styles.Merge(styles.Styles{Background: "theme.danger", Color: "#ffffff", FontSize: 11, BorderRadius: 999}, b.st)
	if b.text == "" {
		st.Width, st.Height = 8, 8
		return elements.Box(st)
	}
	st.Padding = properties.SymmetricInset(5, 1)
	st.MinWidth = 16
	st.Justify = properties.JustifyCenter
	return elements.Row(st, elements.Text(b.text, styles.Styles{Color: st.Color, FontSize: st.FontSize, FontWeight: properties.FontWeightSemiBold}))
}

// Layout lays out the badge, or el with the badge on its corner.
func (b *Badge) Layout(gtx layout.Context) layout.Dimensions {
	if b.target == nil {
		return b.pill().Layout(gtx)
	}
	dims := b.target.Layout(gtx)
	mgtx := gtx
	mgtx.Constraints.Min = image.Point{}
	track := elements.TrackChild()
	macro := op.Record(gtx.Ops)
	bd := b.pill().Layout(mgtx)
	call := macro.Stop()
	track.End()
	// On the target's top end corner: top-right, or top-left right to
	// left.
	pos := image.Pt(dims.Size.X-bd.Size.X*2/3, -bd.Size.Y/3)
	if kit.RTL(gtx) {
		pos.X = -bd.Size.X / 3
	}
	track.Place(pos)
	defer op.Offset(pos).Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
	return dims
}
