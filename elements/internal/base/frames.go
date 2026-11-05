package base

import (
	"sync"
)

// Threading.
//
// The view function, every Layout, scroll content closures and event
// handlers (OnClick, OnChange, ...) all run on one goroutine: the UI
// goroutine. Data changed only there needs no locks. Code running
// elsewhere - a network callback, a timer, an MCP tool - hands work to
// the UI goroutine with App.Post (or App.RunOnUI to wait for it), or
// uses State, which is safe from any goroutine.

var (
	afterMu     sync.Mutex
	afterLayout []func()
)

// AfterLayout runs fn once the current frame's layout is finished, on
// the UI goroutine, then requests another frame. A handler that must
// change the model the rest of this frame is still reading - removing
// the list item being laid out, say - defers the change with it.
// Called outside a frame, fn runs after the next one.
func AfterLayout(fn func()) {
	afterMu.Lock()
	afterLayout = append(afterLayout, fn)
	afterMu.Unlock()
}

// RunAfterLayout runs and clears the AfterLayout queue, reporting
// whether anything ran. The app calls it after laying out each frame;
// component authors don't need to.
func RunAfterLayout() bool {
	afterMu.Lock()
	fns := afterLayout
	afterLayout = nil
	afterMu.Unlock()
	for _, fn := range fns {
		fn()
	}
	if len(fns) > 0 {
		InvalidateCaches()
	}
	return len(fns) > 0
}
