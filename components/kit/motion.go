package kit

import "sync/atomic"

// Reduced motion.
//
// When the user asks the system for less motion (Reduce Motion on
// macOS and iOS, "Show animations" off on Windows, "Remove animations"
// on Android, animations off on GNOME and KDE), Giode's animations
// finish at once: transitions, keyframes, sliding drawers and sheets,
// switches. The app sets it from the system setting or Config.Motion;
// components ask Reduced.

var reducedMotion atomic.Bool

// SetReducedMotion turns reduced motion on or off.
func SetReducedMotion(on bool) { reducedMotion.Store(on) }

// ReducedMotion reports whether animations should finish at once.
func ReducedMotion() bool { return reducedMotion.Load() }

// MotionStep is an animation's step toward its target - a fraction of
// the whole - or the whole of it when motion is reduced.
func MotionStep(step float32) float32 {
	if reducedMotion.Load() {
		return 1
	}
	return step
}
