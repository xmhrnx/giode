package box

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/elements/internal/base"
)

// layoutAbsolute places absolutely positioned Box children at their
// Top/Left (or Right/Bottom) offsets within the parent's content box,
// painting above the flow content of the same z-index. The children do
// not affect the parent's size. With zs set, their drawing is recorded
// into *zs for painting in z order instead of painted.
//
// An offset of 0 is unset; properties.Flush is an explicit 0. Left wins
// over Right and Top over Bottom; with both of a pair set, a child
// without its own size on that axis stretches between them, as in CSS.
func (b *BoxElement) layoutAbsolute(gtx layout.Context, children []base.Element, size image.Point, zs *[]zPaint) {
	for _, child := range children {
		box, ok := child.(*BoxElement)
		if !ok {
			continue
		}
		st := box.st.Px(gtx.Metric)
		left, right := st.Left.Px(), st.Right.Px()
		top, bottom := st.Top.Px(), st.Bottom.Px()

		cgtx := gtx
		cgtx.Constraints.Min = image.Point{}
		cgtx.Constraints.Max = size
		cgtx.Constraints.Max.X -= max(0, left) + max(0, right)
		cgtx.Constraints.Max.Y -= max(0, top) + max(0, bottom)
		cgtx.Constraints.Max.X = max(0, cgtx.Constraints.Max.X)
		cgtx.Constraints.Max.Y = max(0, cgtx.Constraints.Max.Y)
		box.stretchX = st.Left.IsSet() && st.Right.IsSet()
		box.stretchY = st.Top.IsSet() && st.Bottom.IsSet()

		// Left/top-anchored position is known upfront; right/bottom
		// anchoring depends on the child's own size, so it can only be
		// resolved after Layout returns. The absolute-position tracker
		// (see absolute.go) is pushed with the upfront position, and
		// what the child registered is moved by the difference once
		// the final one is known (see ChildTracker).
		upfront := image.Pt(left, top)

		popAbs := base.PushOffset(upfront)
		track := base.TrackChild()
		macro := op.Record(gtx.Ops)
		dims := box.Layout(cgtx)
		box.stretchX, box.stretchY = false, false
		childOps := macro.Stop()
		track.End()
		popAbs()

		pos := upfront
		if !st.Left.IsSet() && st.Right.IsSet() {
			pos.X = size.X - right - dims.Size.X
		}
		if !st.Top.IsSet() && st.Bottom.IsSet() {
			pos.Y = size.Y - bottom - dims.Size.Y
		}
		track.Place(pos.Sub(upfront))

		if zs != nil {
			*zs = append(*zs, zPaint{call: childOps, off: pos, z: base.ZIndexOf(child), positioned: true})
			continue
		}
		off := op.Offset(pos).Push(gtx.Ops)
		childOps.Add(gtx.Ops)
		off.Pop()
	}
}
