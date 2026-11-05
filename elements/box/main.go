package box

import (
	"github.com/xmhrnx/giode/elements/internal/base"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Box is a container element: children are laid along FlexDirection
// with Align, Justify, Gap and Wrap. Chain OnHover to add hover
// styles.
func Box(st styles.Styles, children ...base.Element) *BoxElement {
	return &BoxElement{st: st, children: children}
}

// Row lays children out horizontally. It is Box with flex direction
// forced to row.
func Row(st styles.Styles, children ...base.Element) *BoxElement {
	st.FlexDirection = properties.FlexDirectionRow
	return Box(st, children...)
}

// Stack lays children out vertically. It is Box with flex direction
// forced to column. Note that this is not Gio's layout.Stack.
func Stack(st styles.Styles, children ...base.Element) *BoxElement {
	st.FlexDirection = properties.FlexDirectionColumn
	return Box(st, children...)
}
