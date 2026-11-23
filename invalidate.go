package giode

import "github.com/xmhrnx/giode/internal/platform"

// invalidate asks the window, and the portal windows, for a frame, or
// wakes the UI goroutine while it's hidden. On macOS, Gio's Invalidate
// called on the main thread runs the event loop there and then, holding
// a lock the UI goroutine takes when it invalidates in turn: called from
// an AppKit callback (a Dock click showing the window, the close button
// hiding it, a Services request), the app hung. There it hops to a
// goroutine.
func (a *App) invalidate() {
	if platform.OnMainThread() {
		go a.invalidateAll()
		return
	}
	a.invalidateAll()
}

// invalidateAll asks every window for a frame: a State or the theme
// changed, which any of them may show.
func (a *App) invalidateAll() {
	if a.window != nil {
		a.window.Invalidate()
	}
	a.portalMu.Lock()
	portals := a.portals
	a.portalMu.Unlock()
	for _, p := range portals {
		if p.win != nil {
			p.win.Invalidate()
		}
	}
}
