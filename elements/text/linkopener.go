package text

import "sync/atomic"

// linkOpener opens links no OnLink handler took; the app sets it to
// App.OpenURL.
var linkOpener atomic.Pointer[func(url string)]

// SetLinkOpener sets what opens a Link whose paragraph has no OnLink
// handler. The app sets it to open the browser; component authors
// don't need to.
func SetLinkOpener(fn func(url string)) {
	if fn == nil {
		linkOpener.Store(nil)
		return
	}
	linkOpener.Store(&fn)
}
