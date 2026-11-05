package base

import (
	"gioui.org/io/key"
	"gioui.org/layout"
)

// backEntry is one handler registered for the current frame.
type backEntry struct {
	fn     func()
	escape bool
}

var backStack []backEntry

// HandleBack registers fn to run when the user presses the system Back
// button (Android's Back gesture or key) during this frame. Call it
// from Layout, every frame, for as long as there is somewhere to go
// back to - Router does this while its history isn't empty.
//
// Handlers registered later in the frame win: content laid out on top
// (a dialog over a page) takes Back before what's beneath it, and only
// the topmost handler runs. When no handler is registered, giode
// leaves Back to the platform, which on Android closes the app.
//
// Android asks whether Back is handled the moment it is pressed, using
// the handlers registered in the last frame. That's why handlers are
// registered from Layout rather than consulted afterwards: whether
// Back is consumed must be known before the press arrives.
//
// fn runs after the current frame's layout (see AfterLayout), so it
// may change anything the view reads.
func HandleBack(fn func()) {
	backStack = append(backStack, backEntry{fn: fn})
}

// HandleDismiss is HandleBack for overlays: fn also runs when Escape
// is pressed. Dialog, Drawer and Menu use it while open.
func HandleDismiss(fn func()) {
	backStack = append(backStack, backEntry{fn: fn, escape: true})
}

// DispatchBack delivers this frame's Back and Escape presses to the
// topmost registered handler, then clears the registrations. The app
// calls it once per frame, after the view and portals are laid out and
// before RunAfterLayout; component authors don't need to.
func DispatchBack(gtx layout.Context) {
	entries := backStack
	backStack = nil
	if len(entries) == 0 {
		return
	}
	var back, escape func()
	for _, e := range entries {
		back = e.fn
		if e.escape {
			escape = e.fn
		}
	}
	filters := []key.Filter{{Name: key.NameBack}}
	if escape != nil {
		filters = append(filters, key.Filter{Name: key.NameEscape})
	}
	for _, f := range filters {
		for {
			ev, ok := gtx.Event(f)
			if !ok {
				break
			}
			e, ok := ev.(key.Event)
			if !ok || e.State != key.Press {
				continue
			}
			if e.Name == key.NameBack {
				AfterLayout(back)
			} else {
				AfterLayout(escape)
			}
		}
	}
}
