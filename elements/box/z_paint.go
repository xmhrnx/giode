package box

import (
	"image"
	"sort"

	"gioui.org/op"

	"github.com/xmhrnx/giode/elements/internal/base"
)

// zPaint is a child's recorded drawing and where it goes, painted once
// every child is laid out: in z-index order, positioned children above
// flow children of the same z-index. Later paint also gets input first.
type zPaint struct {
	call       op.CallOp
	off        image.Point
	size       image.Point
	z          int
	positioned bool
	// track holds what the child registered, already placed at off.
	track base.ChildTracker
}

// paintZ paints the recorded children in z order.
func paintZ(ops *op.Ops, items []zPaint) {
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if a.z != b.z {
			return a.z < b.z
		}
		return !a.positioned && b.positioned
	})
	for _, it := range items {
		st := op.Offset(it.off).Push(ops)
		it.call.Add(ops)
		st.Pop()
	}
}
