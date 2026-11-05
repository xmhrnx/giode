package base

import "image"

// PendingPortals returns the portals queued since the last drain,
// without draining them. Element packages and tests inspect the queue;
// callers must not keep or reorder it.
func PendingPortals() []portalEntry {
	return portalQueue
}

// ClearPortals drops every queued portal.
func ClearPortals() {
	portalQueue = nil
}

// Anchor reports the window-absolute point the content is placed near.
func (e portalEntry) Anchor() image.Point {
	return e.anchor
}

// SetAnchor moves the entry's anchor, as ChildTracker does when a
// container places its child after the fact.
func (e *portalEntry) SetAnchor(p image.Point) {
	e.anchor = p
}

// Pos reports where DrainPortals will place the entry.
func (e portalEntry) Pos() image.Point {
	return e.pos()
}
