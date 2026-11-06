package base

import (
	"gioui.org/layout"
	"gioui.org/op"
)

// Measure lays el out without painting, taking input, registering
// portals, Back handlers or selectable text, or recording semantics,
// and returns its size - for components that must know a child's size
// before placing it.
func Measure(gtx layout.Context, el Element) layout.Dimensions {
	scratch := gtx
	scratch.Ops = new(op.Ops)
	scratch.Source = gtx.Source.Disabled()
	saved := semantics
	semantics = nil
	portals, backs, nprobes := len(portalQueue), len(backStack), len(probes)
	measuring++
	dims := el.Layout(scratch)
	measuring--
	portalQueue = portalQueue[:portals]
	backStack = backStack[:backs]
	probes = probes[:nprobes]
	semantics = saved
	return dims
}

// measuring counts the Measure calls in progress.
var measuring int

// Measuring reports whether the element being laid out is only being
// measured: what it registers for the frame - selectable text, say -
// is dropped with its ops, so it registers nothing.
func Measuring() bool { return measuring > 0 }
