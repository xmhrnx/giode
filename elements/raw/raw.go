package raw

import (
	"gioui.org/layout"

	"github.com/xmhrnx/giode/elements/internal/base"
)

// RawFunc adapts a function to the Element interface.
type RawFunc func(gtx layout.Context) layout.Dimensions

// Layout implements Element.
func (f RawFunc) Layout(gtx layout.Context) layout.Dimensions {
	return f(gtx)
}

// Raw drops down to plain Gio code at any point in the element tree.
// Low-level customization stays possible everywhere.
func Raw(fn RawFunc) base.Element {
	return fn
}
