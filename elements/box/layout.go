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
//
// A box with OnHover or OnClick is kept and laid out every frame as the
// same element; when all its content is CacheSafe it replays the output
// recorded last time until its hover or press state, its content's
// state, its constraints or its direction change.
func (b *BoxElement) Layout(gtx layout.Context) layout.Dimensions {
	if (b.hover != nil || b.press != nil) && b.CacheSafe() {
		if b.cache == nil {
			b.cache = base.NewMemoCache()
			b.cached = boxUncached{b}
		}
		return b.cache.LayoutOwn(gtx, b.cachedBuild, noWatch, b.stretchX, b.stretchY)
	}
	return b.layoutUncached(gtx)
}

func (b *BoxElement) cachedBuild() base.Element { return b.cached }

// noWatch is the watch of a box: its fields don't change once built.
func noWatch() []any { return nil }

// boxUncached lays the box out without its cache.
type boxUncached struct{ b *BoxElement }

func (u boxUncached) Layout(gtx layout.Context) layout.Dimensions { return u.b.layoutUncached(gtx) }

func (b *BoxElement) layoutUncached(gtx layout.Context) layout.Dimensions {
	if b.press != nil {
		// OnClick: the Pressable lays the box out over its hit area.
		b.press.cursor = b.st.Cursor
		b.press.content = func(PressState) base.Element { return boxBody{b} }
		return b.press.Layout(gtx)
	}
	return b.layoutBody(gtx)
}

func (bb boxBody) Layout(gtx layout.Context) layout.Dimensions { return bb.b.layoutBody(gtx) }
