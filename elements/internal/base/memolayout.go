package base

import (
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"

	"github.com/xmhrnx/giode/components/kit"
)

// Layout lays out build's element, or replays the cached result when
// nothing it depends on changed.
func (c *MemoCache) Layout(gtx layout.Context, build func() Element, deps ...any) layout.Dimensions {
	if c.frame == frameNumber && frameNumber != 0 {
		// Already used this frame: its ops are referenced by the
		// earlier use and must not be re-recorded.
		return build().Layout(gtx)
	}
	c.frame = frameNumber

	// A subtree with input handling but no passes is laid out live
	// while the pointer is active over it, so those widgets see their
	// events.
	active := false
	if c.input.Opaque {
		for {
			ev, ok := gtx.Event(pointer.Filter{
				Target: c,
				Kinds:  pointer.Enter | pointer.Leave | pointer.Move | pointer.Press | pointer.Release | pointer.Drag | pointer.Cancel,
			})
			if !ok {
				break
			}
			pe, ok := ev.(pointer.Event)
			if !ok {
				continue
			}
			active = true
			switch pe.Kind {
			case pointer.Leave, pointer.Cancel:
				c.inside = false
			default:
				c.inside = true
			}
		}
	}

	reuse := c.fresh(gtx, deps) && !(c.input.Opaque && (active || c.inside))
	// Deliver input to the widgets inside without laying them out;
	// rebuild only if one changed.
	if reuse && len(c.input.Passes) > 0 && kit.RunPasses(gtx, c.input.Passes) {
		reuse = false
	}

	if reuse {
		c.hits++
		c.keepAlive()
		origin := AbsolutePosition()
		for i, p := range c.probes {
			*p = origin.Add(c.probeRel[i])
			probes = append(probes, p)
		}
		if semantics != nil {
			for _, n := range c.nodes {
				n.Bounds = n.Bounds.Add(origin)
				*semantics = append(*semantics, n)
			}
		}
		c.call.Add(gtx.Ops)
	} else {
		c.misses++
		c.record(gtx, build, deps)
	}
	c.dirty = false

	// Enclosing caches replaying this one run its passes and keep its
	// state alive through it.
	if c.input.Opaque {
		kit.OpaqueInput()
	} else {
		// Registered even without passes of its own: the pass also
		// notices state the subtree reads changing.
		if c.passFn == nil {
			c.passFn = c.pass
		}
		kit.HandleInput(c.passFn)
	}
	retainKeyed(c.keepAlive)

	if c.input.Opaque {
		// Observe the pointer over the subtree without blocking
		// anything beneath it.
		ps := pointer.PassOp{}.Push(gtx.Ops)
		cl := clip.Rect{Max: c.dims.Size}.Push(gtx.Ops)
		event.Op(gtx.Ops, c)
		cl.Pop()
		ps.Pop()
	}
	return c.dims
}
