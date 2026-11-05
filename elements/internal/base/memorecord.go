package base

import (
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/state"
	"github.com/xmhrnx/giode/styles/properties"
)

// fresh reports whether the recorded output is still valid for gtx and
// deps, input aside.
func (c *MemoCache) fresh(gtx layout.Context, deps []any) bool {
	return c.valid && !c.live && !c.portal && !c.dirty &&
		c.constraints == gtx.Constraints &&
		c.metric == gtx.Metric &&
		c.theme == properties.ThemeVersion() &&
		c.epoch == cacheEpoch.Load() &&
		c.color == CurrentColor() &&
		c.breakpoint == ActiveBreakpoint() &&
		c.touch == TouchInput() &&
		c.locale == gtx.Locale &&
		c.selection == SelectionVersion() &&
		depsEqual(c.deps, deps) &&
		!c.reads.Changed()
}

// pass is the cache's input pass, run by an enclosing cache replaying
// it: it delivers input to the widgets inside, and reports a change
// when they changed or the cache must rebuild anyway.
func (c *MemoCache) pass(gtx layout.Context) bool {
	if !c.valid || c.live || c.reads.Changed() || c.epoch != cacheEpoch.Load() || c.theme != properties.ThemeVersion() ||
		(c.watch != nil && !depsEqual(c.watched, c.watch())) {
		c.dirty = true
		return true
	}
	if kit.RunPasses(gtx, c.input.Passes) {
		c.dirty = true
		return true
	}
	return false
}

// keepAlive refreshes the keyed state the subtree looked up when it was
// recorded.
func (c *MemoCache) keepAlive() {
	c.alive = frameNumber
	for _, f := range c.retain {
		f()
	}
}

// record rebuilds and lays out the subtree into the cache's own ops.
func (c *MemoCache) record(gtx layout.Context, build func() Element, deps []any) {
	c.ops.Reset()
	cgtx := gtx
	cgtx.Ops = &c.ops

	c.live = false
	popLive := kit.PushLiveScope(&c.live)
	c.input = kit.InputScope{Passes: c.input.Passes[:0]}
	popInput := kit.PushInputScope(&c.input)
	c.retain = c.retain[:0]
	popRetain := pushRetainScope(&c.retain)
	reads := state.Track()
	portals, backs, probeStart := len(portalQueue), len(backStack), len(probes)
	semStart := 0
	if semantics != nil {
		semStart = len(*semantics)
	}

	macro := op.Record(&c.ops)
	c.dims = build().Layout(cgtx)
	c.call = macro.Stop()

	reads.Stop()
	popRetain()
	popInput()
	popLive()
	c.alive = frameNumber

	c.reads = reads
	c.portal = len(portalQueue) > portals || len(backStack) > backs
	origin := AbsolutePosition()
	c.probes, c.probeRel = c.probes[:0], c.probeRel[:0]
	for _, p := range probes[probeStart:] {
		c.probes = append(c.probes, p)
		c.probeRel = append(c.probeRel, p.Sub(origin))
	}
	c.nodes = c.nodes[:0]
	if semantics != nil {
		for _, n := range (*semantics)[semStart:] {
			n.Bounds = n.Bounds.Sub(origin)
			c.nodes = append(c.nodes, n)
		}
	}
	c.constraints = gtx.Constraints
	c.metric = gtx.Metric
	c.theme = properties.ThemeVersion()
	c.epoch = cacheEpoch.Load()
	c.color = CurrentColor()
	c.breakpoint = ActiveBreakpoint()
	c.touch = TouchInput()
	c.locale = gtx.Locale
	c.selection = SelectionVersion()
	c.deps = append(c.deps[:0], deps...)
	c.valid = true

	c.call.Add(gtx.Ops)
}
