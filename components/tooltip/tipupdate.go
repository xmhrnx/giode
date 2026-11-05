// Package tooltip provides a stateful tooltip, shown on hover or
// long-press, that wraps any element as its trigger.

package tooltip

import (
	"time"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"github.com/xmhrnx/giode/components/kit"
)

// update processes the pointer events queued on the trigger's area.
func (t *Tooltip) update(gtx layout.Context) {
	for {
		ev, ok := gtx.Event(pointer.Filter{
			Target: t,
			Kinds:  pointer.Enter | pointer.Leave | pointer.Press | pointer.Release | pointer.Cancel,
		})
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		touch := e.Source == pointer.Touch
		switch e.Kind {
		case pointer.Enter:
			if !touch {
				t.hovered = true
			}
		case pointer.Leave:
			if !touch {
				t.hovered = false
			} else {
				// The finger slid off before (or after) the long-press.
				t.touchAt, t.longPressed = time.Time{}, false
			}
		case pointer.Cancel:
			// A scroll took the pointer, or the window lost it.
			t.hovered, t.touchAt, t.longPressed = false, time.Time{}, false
		case pointer.Press:
			if touch {
				t.touchAt = gtx.Now
				t.shownUntil = time.Time{}
				kit.RequestFrameAt(gtx, gtx.Now.Add(longPress))
			}
		case pointer.Release:
			if touch {
				if t.longPressed {
					t.shownUntil = gtx.Now.Add(linger)
				}
				t.touchAt, t.longPressed = time.Time{}, false
			}
		}
	}
	if !t.touchAt.IsZero() && !t.longPressed {
		if gtx.Now.Sub(t.touchAt) >= longPress {
			t.longPressed = true
		} else {
			kit.RequestFrameAt(gtx, t.touchAt.Add(longPress))
		}
	}
	if !t.shownUntil.IsZero() {
		if gtx.Now.Before(t.shownUntil) {
			kit.RequestFrameAt(gtx, t.shownUntil)
		} else {
			t.shownUntil = time.Time{}
		}
	}
}
