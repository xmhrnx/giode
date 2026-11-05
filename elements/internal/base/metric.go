package base

import (
	"math"
	"sync/atomic"

	"gioui.org/unit"
)

// displayScale is the last frame's pixels per dp, stored as float32
// bits so it can be read from any goroutine.
var displayScale atomic.Uint32

func init() {
	displayScale.Store(math.Float32bits(1))
}

// SetMetric records the frame's display metric. The app calls it every
// frame; component authors don't need to.
func SetMetric(m unit.Metric) {
	s := m.PxPerDp
	if s <= 0 {
		s = 1
	}
	displayScale.Store(math.Float32bits(s))
}

// DisplayScale returns the display's pixels per dp as of the last
// frame (1 on a standard display, 2 on a typical HiDPI one). It is
// safe to call from anywhere, including outside Layout, to convert
// sizes for code that doesn't have a layout context.
func DisplayScale() float32 {
	return math.Float32frombits(displayScale.Load())
}

// DpToPx converts dp to pixels with the last frame's display scale.
func DpToPx(v float32) int {
	return int(math.Round(float64(v * DisplayScale())))
}

// PxToDp converts pixels to dp with the last frame's display scale.
func PxToDp(px int) float32 {
	return float32(px) / DisplayScale()
}
