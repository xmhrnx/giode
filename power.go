package giode

// powerEvent runs OnSuspend or OnResume on the UI goroutine and waits
// for it (Linux holds the sleep until it returns).
func (a *App) powerEvent(suspend bool) {
	f := a.cfg.OnResume
	if suspend {
		f = a.cfg.OnSuspend
	}
	if f == nil {
		return
	}
	done := make(chan struct{})
	a.Post(func() {
		defer close(done)
		f()
	})
	<-done
}
