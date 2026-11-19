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

// Grid lays children out on a grid: Box with Display DisplayGrid. st's
// GridColumns sizes the columns ("200 1fr", "repeat(3, 1fr)",
// "repeat(auto-fill, minmax(220, 1fr))"), Gap and RowGap space them, and
// each child's GridColumn and GridRow place it (Span, Cell), or it takes
// the next free cells in reading order.
func Grid(st styles.Styles, children ...base.Element) *BoxElement {
	st.Display = properties.DisplayGrid
	return Box(st, children...)
}

// Stack lays children out vertically. It is Box with flex direction
// forced to column. Note that this is not Gio's layout.Stack.
func Stack(st styles.Styles, children ...base.Element) *BoxElement {
	st.FlexDirection = properties.FlexDirectionColumn
	return Box(st, children...)
}
