package box

import (
	"strconv"
	"sync/atomic"
	"time"
)

// LongPressDuration is how long a touch must be held still for
// OnLongPress.
const LongPressDuration = 500 * time.Millisecond

// Describe names the pressable for diagnostics.
func (p *Pressable) Describe() string {
	kind := "Pressable"
	if p.label != "" {
		return kind + " " + strconv.Quote(p.label)
	}
	return kind
}

// longPressFeedback plays when a long-press fires; the app sets it to
// haptic feedback.
var longPressFeedback atomic.Pointer[func()]

// SetLongPressFeedback sets what plays when OnLongPress fires - the app
// sets it to a haptic tap; component authors don't need to.
func SetLongPressFeedback(fn func()) {
	if fn == nil {
		longPressFeedback.Store(nil)
		return
	}
	longPressFeedback.Store(&fn)
}
