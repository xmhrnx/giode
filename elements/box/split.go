package box

import (
	"github.com/xmhrnx/giode/elements/internal/base"
	"github.com/xmhrnx/giode/styles/properties"
)

// split partitions children into flow and absolutely positioned
// children, in document order, and reports whether any child sets a
// z-index - then the children are painted in z order (see zPaint)
// rather than as laid out. Absolute positioning applies to Box
// children.
func (b *BoxElement) split() (flow, absolute, outside []base.Element, ordered bool) {
	// Fast path, the overwhelmingly common case: no absolute children
	// and no z-index, so the children are used as they are without
	// allocating.
	simple := true
	for _, child := range b.children {
		if base.ZIndexOf(child) != 0 {
			simple, ordered = false, true
		}
		if box, ok := child.(*BoxElement); ok && box.st.Position.Mode() == properties.PositionAbsolute {
			simple = false
		}
		if _, ok := child.(base.OutOfFlow); ok {
			simple = false
		}
	}
	if simple {
		return b.children, nil, nil, false
	}
	flow = make([]base.Element, 0, len(b.children))
	for _, child := range b.children {
		if _, ok := child.(base.OutOfFlow); ok {
			outside = append(outside, child)
		} else if box, ok := child.(*BoxElement); ok && box.st.Position.Mode() == properties.PositionAbsolute {
			absolute = append(absolute, child)
		} else {
			flow = append(flow, child)
		}
	}
	return flow, absolute, outside, ordered
}
