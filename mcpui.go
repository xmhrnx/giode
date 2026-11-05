package giode

import (
	"context"
	"fmt"

	"github.com/xmhrnx/giode/elements"
)

// Post queues fn to run on the UI goroutine before the next frame and
// requests that frame, without waiting. While the window is hidden or
// minimized, and there are no frames, it runs as soon as the UI
// goroutine wakes up. It is safe to call from any
// goroutine: use it to apply results from background work (a network
// response, a file read) to data the view reads, so that data never
// needs locks. Calls run in the order posted.
func (a *App) Post(fn func()) {
	a.uiMu.Lock()
	a.uiCalls = append(a.uiCalls, fn)
	a.uiMu.Unlock()
	a.window.Invalidate()
}

// RunOnUI runs fn on the UI goroutine before the next frame is laid
// out (or straight away while the window is hidden) and waits for it,
// so code running elsewhere (an MCP tool, a network callback) can
// safely drive stateful components. It fails if the UI goroutine
// doesn't run it before ctx is done.
func (a *App) RunOnUI(ctx context.Context, fn func()) error {
	done := make(chan struct{})
	a.uiMu.Lock()
	a.uiCalls = append(a.uiCalls, func() { fn(); close(done) })
	a.uiMu.Unlock()
	a.window.Invalidate()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("giode: UI thread didn't run the call (window closed or busy?): %w", ctx.Err())
	}
}

// runUICalls runs the queued Post and RunOnUI calls, reporting whether
// there were any. The frame loop calls it before laying out, and on
// the wakeups Invalidate causes when no frame comes (a hidden or
// minimized window), so tray menus and other callers still reach the
// UI goroutine.
func (a *App) runUICalls() bool {
	a.uiMu.Lock()
	calls := a.uiCalls
	a.uiCalls = nil
	a.uiMu.Unlock()
	if len(calls) == 0 {
		return false
	}
	for _, fn := range calls {
		fn()
	}
	// Calls usually drive components programmatically; make sure
	// cached subtrees pick that up.
	elements.InvalidateCaches()
	return true
}
