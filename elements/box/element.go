package box

import (
	"gioui.org/gesture"
	"gioui.org/layout"

	"github.com/xmhrnx/giode/elements/internal/base"
	"github.com/xmhrnx/giode/styles"
)

// BoxElement is a container element: a flex container and a
// containing block for absolutely positioned children.
type BoxElement struct {
	st       styles.Styles
	children []base.Element

	// Hover styles, set with OnHover. The hovered field caches the
	// last observed hover state.
	hover   *gesture.Hover
	hoverSt styles.Styles
	hovered bool
	// hoverPass is the hover input pass (see kit.HandleInput).
	hoverPass func(gtx layout.Context) bool
	hoverAnim

	// press makes the box clickable; see OnClick.
	press *Pressable

	// stretchX and stretchY are set by a parent laying the box out
	// absolutely between Left and Right (Top and Bottom): it fills the
	// constraints on that axis.
	stretchX, stretchY bool

	// cache holds the output of a box laid out every frame as the same
	// element (one with OnHover or OnClick, which must be kept), when
	// everything in it is CacheSafe; see Layout. safe caches that
	// verdict: 0 unknown, 1 safe, -1 not.
	cache  *base.MemoCache
	safe   int8
	cached base.Element
}

// CacheSafe reports whether the box's output depends only on things a
// cache watches: its own styles, which don't change after it's built,
// and children that are CacheSafe themselves.
func (b *BoxElement) CacheSafe() bool {
	if b.safe == 0 {
		b.safe = 1
		for _, c := range b.children {
			if !base.IsCacheSafe(c) {
				b.safe = -1
				break
			}
		}
	}
	return b.safe == 1
}

func (b *BoxElement) FlexWeight(axis layout.Axis) int {
	return base.WeightOf(b.st, axis)
}

func (b *BoxElement) FlexPercent(axis layout.Axis) int {
	return base.PercentOf(b.st, axis)
}

func (b *BoxElement) ZIndex() int {
	return int(b.st.ZIndex)
}
