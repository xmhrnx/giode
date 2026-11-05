package base

import (
	"image"

	"gioui.org/io/system"
	"gioui.org/layout"
)

// RegisterPortal queues build to be laid out and painted for real
// after the rest of the view - the mechanism behind Tooltip, nested
// Menu submenus and AutoComplete's suggestion panel, and the building
// block for any component that needs to render on top of everything
// else no matter how deeply it's nested in the tree.
//
// anchor is a window-absolute point (typically AbsolutePosition(),
// optionally offset - e.g. by the trigger element's own height) that
// the content is positioned near; it's first laid out once on a
// disabled scratch context, bounded by the window (see
// SetWindowSize) rather than gtx's own ambient constraints, to
// measure its size (the same technique Menu's own panel-width
// measurement uses) so the position can be clamped to stay inside the
// window, flipping above the anchor if it would overflow the bottom
// edge - see kit.ClampPos. build is then laid out for real, exactly
// once, when the queue is drained.
//
// The anchor follows the caller when an enclosing Row, Stack or Scroll
// places it after the fact (see ChildTracker), so a portal opened from
// one of several children of a row lands below that child, and the
// clamping happens once its final position is known.
func RegisterPortal(gtx layout.Context, anchor image.Point, build func() Element) {
	win := windowSize
	if win == (image.Point{}) {
		win = gtx.Constraints.Max
	}

	mgtx := gtx
	mgtx.Constraints = layout.Constraints{Max: win}
	dims := Measure(mgtx, build())

	portalQueue = append(portalQueue, portalEntry{anchor: anchor, registered: anchor, size: dims.Size, win: win, build: build})
}

// RegisterPortalAt queues build like RegisterPortal at anchor, a
// window position that stays put: containers that place their children
// after laying them out don't move it, unlike RegisterPortal's anchor.
// For content opened at a known window position - a context menu at the
// pointer. With alignEnd, anchor is the content's top-right corner.
func RegisterPortalAt(gtx layout.Context, anchor image.Point, alignEnd bool, build func() Element) {
	RegisterPortal(gtx, anchor, build)
	e := &portalQueue[len(portalQueue)-1]
	e.fixed = true
	if alignEnd {
		e.anchor.X -= e.size.X
	}
}

// RegisterPortalAround queues build like RegisterPortal, placed next to
// trigger, a window-absolute rectangle (AbsolutePosition() and the
// trigger's size): gap pixels below it or above it, per place. Unlike
// RegisterPortal's flip, which only knows a point, flipping around the
// rectangle never covers the trigger - a tooltip on a control bar at
// the bottom of the window goes above its button.
//
// Its start edges line up with the trigger's: left edges, or right
// edges when gtx reads right to left.
func RegisterPortalAround(gtx layout.Context, trigger image.Rectangle, gap int, place Placement, build func() Element) {
	RegisterPortal(gtx, trigger.Min, build)
	e := &portalQueue[len(portalQueue)-1]
	e.around = trigger.Sub(trigger.Min)
	e.hasAround, e.gap, e.place = true, gap, place
	e.alignEnd = gtx.Locale.Direction == system.RTL
}

// RegisterPortalEnd queues build like RegisterPortal with anchor as the
// content's top-right corner rather than its top-left: a menu opened
// at a point in a right-to-left layout extends to the left of it.
func RegisterPortalEnd(gtx layout.Context, anchor image.Point, build func() Element) {
	RegisterPortal(gtx, anchor, build)
	e := &portalQueue[len(portalQueue)-1]
	e.anchor.X -= e.size.X
}

// RegisterOverlay queues build to be laid out over the whole window,
// above the rest of the view, like a portal anchored at the window's
// origin that's neither measured nor clamped: build gets the window's
// size as its maximum constraints and positions its own content - a
// bottom sheet, a full-screen picker. Overlays and portals paint in
// registration order.
func RegisterOverlay(build func() Element) {
	portalQueue = append(portalQueue, portalEntry{build: build, overlay: true})
}
