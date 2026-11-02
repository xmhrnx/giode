package elements

import (
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Stack lays children out vertically. It is Box with flex direction
// forced to column. Note that this is not Gio's layout.Stack.
func Stack(st styles.Styles, children ...Element) *BoxElement {
	st.FlexDirection = properties.FlexDirectionColumn
	return Box(st, children...)
}
