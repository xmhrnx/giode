package box

import (
	"gioui.org/layout"

	"github.com/xmhrnx/giode/elements/internal/base"
)

// boxBody lays out a clickable box's own content inside its Pressable.
type boxBody struct{ b *BoxElement }

// Layout resolves the box size and paints the background stack
// (shadow, fill, gradient, image, border) with the content replayed on
// top.
func (b *BoxElement) Layout(gtx layout.Context) layout.Dimensions {
	if b.press != nil {
		// OnClick: the Pressable lays the box out over its hit area.
		b.press.cursor = b.st.Cursor
		b.press.content = func(PressState) base.Element { return boxBody{b} }
		return b.press.Layout(gtx)
	}
	return b.layoutBody(gtx)
}

func (bb boxBody) Layout(gtx layout.Context) layout.Dimensions { return bb.b.layoutBody(gtx) }
