package giode

// vetoClose stops a close the user asked for when Config.OnCloseRequest
// is set and reports true; the handler then decides on the UI
// goroutine, closing (or hiding to the tray) when it says so. Closes the
// app makes itself (Close, Quit) aren't asked about.
func (a *App) vetoClose() bool {
	if a.quitting.Load() {
		return false
	}
	if a.cfg.NoClose {
		return true
	}
	if a.cfg.OnCloseRequest == nil {
		return false
	}
	if a.closeAsked.Swap(true) {
		return true // already asking
	}
	a.Post(func() {
		ok := a.cfg.OnCloseRequest()
		a.closeAsked.Store(false)
		if !ok {
			return
		}
		if a.closesToTray() {
			a.HideWindow()
			return
		}
		a.Close()
	})
	return true
}
