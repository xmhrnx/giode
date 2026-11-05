package base

import (
	"image"
	"runtime"
	"sync/atomic"

	"gioui.org/layout"
	"gioui.org/op"
)

// touchInput records whether the user is interacting by touch. It
// starts true on phones and tablets and follows the last press
// elsewhere (a laptop with a touch screen), as reported by the app.
var touchInput atomic.Bool

func init() {
	touchInput.Store(runtime.GOOS == "android" || runtime.GOOS == "ios")
}

// TouchInput reports whether the user is interacting by touch: always
// on Android and iOS until a mouse is used, and on other platforms
// after a touch press. Components use it to grow their touch targets
// (see MinTouchTarget).
func TouchInput() bool {
	return touchInput.Load()
}

// SetTouchInput records whether the last press came from a touch
// screen and reports whether that changed. The app calls it for every
// press; component authors don't need to.
func SetTouchInput(v bool) (changed bool) {
	return touchInput.Swap(v) != v
}

// minTouchTarget is the smallest comfortable touch target, in dp,
// following the Android guideline.
const minTouchTarget = 48

// MinTouchTarget lays out w and, while TouchInput is on, grows the
// returned size to at least 48x48dp (bounded by the constraints) with
// w centered inside it. Wrap a control's visual in it, inside the
// widget that takes its clicks, so the hit area grows while the
// visual keeps its size:
//
//	dims := clickable.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
//		return elements.MinTouchTarget(gtx, visual)
//	})
//
// With a mouse it lays w out unchanged.
func MinTouchTarget(gtx layout.Context, w layout.Widget) layout.Dimensions {
	if !TouchInput() {
		return w(gtx)
	}
	target := gtx.Constraints.Constrain(image.Pt(gtx.Dp(minTouchTarget), gtx.Dp(minTouchTarget)))
	track := TrackChild()
	macro := op.Record(gtx.Ops)
	dims := w(gtx)
	call := macro.Stop()
	track.End()
	size := image.Pt(max(dims.Size.X, target.X), max(dims.Size.Y, target.Y))
	if size == dims.Size {
		call.Add(gtx.Ops)
		return dims
	}
	off := size.Sub(dims.Size).Div(2)
	track.Place(off)
	defer op.Offset(off).Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
	return layout.Dimensions{Size: size, Baseline: dims.Baseline + size.Y - dims.Size.Y - off.Y}
}
