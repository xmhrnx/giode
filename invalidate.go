package giode

// invalidate asks the window for a frame, or wakes the UI goroutine
// while it's hidden. On macOS, Gio's Invalidate called on the main
// thread runs the event loop there and then, holding a lock the UI
// goroutine takes when it invalidates in turn: called from an AppKit
// callback (a Dock click showing the window, the close button hiding
// it, a Services request), the app hung. There it hops to a goroutine.
func (a *App) invalidate() {
	if onMainThread() {
		go a.window.Invalidate()
		return
	}
	a.window.Invalidate()
}
