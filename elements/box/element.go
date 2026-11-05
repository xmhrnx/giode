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
