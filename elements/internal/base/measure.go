package base

import (
	"gioui.org/layout"
	"gioui.org/op"
)

// Measure lays el out without painting, taking input, registering
// portals or Back handlers, or recording semantics, and returns its size - for
// components that must know a child's size before placing it.
func Measure(gtx layout.Context, el Element) layout.Dimensions {
	scratch := gtx
	scratch.Ops = new(op.Ops)
	scratch.Source = gtx.Source.Disabled()
	saved := semantics
	semantics = nil
	portals, backs, nprobes := len(portalQueue), len(backStack), len(probes)
	dims := el.Layout(scratch)
	portalQueue = portalQueue[:portals]
	backStack = backStack[:backs]
	probes = probes[:nprobes]
	semantics = saved
	return dims
}
