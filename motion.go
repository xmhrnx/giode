package giode

import (
	"time"

	"github.com/xmhrnx/giode/components/kit"
)

// MotionPreference is whether animations play (Config.Motion).
type MotionPreference int

const (
	// MotionSystem follows the system's reduce-motion setting.
	MotionSystem MotionPreference = iota
	// MotionReduced finishes animations at once.
	MotionReduced
	// MotionFull plays them whatever the system says.
	MotionFull
)

// ReducedMotion reports whether animations finish at once: the user
// asked the system for less motion (Reduce Motion on macOS and iOS,
// "Show animations" off on Windows, "Remove animations" on Android,
// animations off on GNOME and KDE, prefers-reduced-motion on the web),
// or Config.Motion or SetMotion said so. Custom animations should
// check it.
func ReducedMotion() bool { return kit.ReducedMotion() }

// SetMotion overrides Config.Motion while the app runs.
func (a *App) SetMotion(p MotionPreference) {
	a.motion.Store(int32(p))
	switch p {
	case MotionReduced:
		kit.SetReducedMotion(true)
	case MotionFull:
		kit.SetReducedMotion(false)
	default:
		if v, err := readReducedMotion(); err == nil {
			kit.SetReducedMotion(v)
		}
	}
	a.invalidate()
}

// SystemReducedMotion reports the system's reduce-motion setting, or
// ErrNotSupported where it can't be read.
func SystemReducedMotion() (bool, error) { return readReducedMotion() }

// readReducedMotion is systemReducedMotion, replaceable in tests.
var readReducedMotion = systemReducedMotion

// watchMotion applies the reduce-motion setting at startup and when it
// changes: checked every few seconds and when the window regains focus.
func (a *App) watchMotion(done <-chan struct{}) {
	a.SetMotion(a.cfg.Motion)
	go func() {
		t := time.NewTicker(themePoll)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
			case <-a.motionCheck:
			}
			if MotionPreference(a.motion.Load()) != MotionSystem {
				continue
			}
			v, err := readReducedMotion()
			if err != nil || v == kit.ReducedMotion() {
				continue
			}
			kit.SetReducedMotion(v)
			a.invalidate()
		}
	}()
}
