package base

import (
	"image"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/components/kit"
)

// pos clamps the entry's anchor to the window.
func (e portalEntry) pos() image.Point {
	if e.overlay {
		return image.Point{}
	}
	if e.hasAround {
		return e.aroundPos()
	}
	winF := f32.Pt(float32(e.win.X), float32(e.win.Y))
	size := f32.Pt(float32(e.size.X), float32(e.size.Y))
	return kit.ClampPos(winF, size, f32.Pt(float32(e.anchor.X), float32(e.anchor.Y)), kit.Margin).Round()
}

// DrainPortals lays out and paints every portal registered so far
// this frame, in registration order (later registrations paint on
// top of earlier ones), then clears the queue. The app calls this
// once per frame, after the rest of the view.
//
// AbsolutePosition/PushOffset are kept in sync with the paint offset
// here (not just op.Offset) so that content built inside a portal -
// nested Menu submenus register a further portal of their own,
// anchored off their own row's position - reports its true screen
// position rather than whatever was accumulated before the portal
// that contains it was originally triggered.
//
// Portals registered while draining (a submenu opened by its parent
// menu's row) are drained too, in the same frame.
func DrainPortals(gtx layout.Context) {
	for len(portalQueue) > 0 {
		entries := portalQueue
		portalQueue = nil
		for _, e := range entries {
			if e.hasAround && e.anchor != e.registered {
				mgtx := gtx
				mgtx.Constraints = layout.Constraints{Max: e.win}
				e.size = Measure(mgtx, e.build()).Size
			}
			pos := e.pos()
			off := op.Offset(pos).Push(gtx.Ops)
			pop := PushOffset(pos)
			e.build().Layout(gtx)
			pop()
			off.Pop()
		}
	}
}

// aroundPos places a RegisterPortalAround entry beside its trigger.
func (e portalEntry) aroundPos() image.Point {
	m := int(kit.Margin)
	r := e.around.Add(e.anchor)
	below := r.Max.Y + e.gap
	above := r.Min.Y - e.gap - e.size.Y
	fitsBelow := below+e.size.Y+m <= e.win.Y
	fitsAbove := above >= m
	y := below
	switch e.place {
	case PlaceAbove:
		y = above
	case PlaceAuto:
		if !fitsBelow && (fitsAbove || r.Min.Y > e.win.Y-r.Max.Y) {
			y = above
		}
	}
	// Keep it in the window: along the side it's on, it may then
	// overlap the trigger, but only when the window is too small for
	// either side.
	y = max(m, min(y, e.win.Y-e.size.Y-m))
	x := r.Min.X
	if e.alignEnd {
		x = r.Max.X - e.size.X
	}
	x = max(m, min(x, e.win.X-e.size.X-m))
	return image.Pt(x, y)
}
