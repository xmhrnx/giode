// Package base provides the shared runtime every element package is
// built on: the Element interface, the frame and color helpers, the
// absolute-position, portal, semantic and Back/OnKey registries, and
// the keyed and memo caches. Element packages import it; users of the
// library import the elements facade instead.
package base

import "gioui.org/layout"

// Element is anything that can lay itself out in a frame.
type Element interface {
	Layout(gtx layout.Context) layout.Dimensions
}
