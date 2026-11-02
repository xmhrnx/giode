package elements

import (
	"image"
	"sort"

	"gioui.org/gesture"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// BoxElement is a container element: a flex container and a
// containing block for absolutely positioned children.
type BoxElement struct {
	st       styles.Styles
	children []Element

	// Hover styles, set with OnHover. The hovered field caches the
	// last observed hover state.
	hover   *gesture.Hover
	hoverSt styles.Styles
	hovered bool
	// hoverPass is the hover input pass (see kit.HandleInput).
	hoverPass func(gtx layout.Context) bool
	hoverAnim

	// press makes the box clickable; see OnClick.
	press *Pressable

	// stretchX and stretchY are set by a parent laying the box out
	// absolutely between Left and Right (Top and Bottom): it fills the
	// constraints on that axis.
	stretchX, stretchY bool
}

// Box is a container element: children are laid along FlexDirection
// with Align, Justify, Gap and Wrap. Chain OnHover to add hover
// styles.
func Box(st styles.Styles, children ...Element) *BoxElement {
	return &BoxElement{st: st, children: children}
}

// flexible marks elements that carry layout hints: how much they
// grow on the flex main axis, their percentage size and their
// z-index.
type flexible interface {
	flexWeight(axis layout.Axis) int
	percentOf(axis layout.Axis) int
	zIndex() int
}

func (b *BoxElement) flexWeight(axis layout.Axis) int {
	return weightOf(b.st, axis)
}

func (b *BoxElement) percentOf(axis layout.Axis) int {
	return percentOf(b.st, axis)
}

func (b *BoxElement) zIndex() int {
	return int(b.st.ZIndex)
}

func weightOf(st styles.Styles, axis layout.Axis) int {
	if st.FlexGrow > 0 {
		return int(st.FlexGrow)
	}
	if axis == layout.Horizontal && st.Width == -1 {
		return 1
	}
	if axis == layout.Vertical && st.Height == -1 {
		return 1
	}
	return 0
}

func percentOf(st styles.Styles, axis layout.Axis) int {
	if axis == layout.Horizontal {
		return int(st.WidthPct)
	}
	return int(st.HeightPct)
}

func flexWeightOf(el Element, axis layout.Axis) int {
	if f, ok := el.(flexible); ok {
		return f.flexWeight(axis)
	}
	return 0
}

func flexPercentOf(el Element, axis layout.Axis) int {
	if f, ok := el.(flexible); ok {
		return f.percentOf(axis)
	}
	return 0
}

func zIndexOf(el Element) int {
	if f, ok := el.(flexible); ok {
		return f.zIndex()
	}
	return 0
}

// split partitions children into flow and absolutely positioned
// children, in document order, and reports whether any child sets a
// z-index - then the children are painted in z order (see zPaint)
// rather than as laid out. Absolute positioning applies to Box
// children.
func (b *BoxElement) split() (flow, absolute, outside []Element, ordered bool) {
	// Fast path, the overwhelmingly common case: no absolute children
	// and no z-index, so the children are used as they are without
	// allocating.
	simple := true
	for _, child := range b.children {
		if zIndexOf(child) != 0 {
			simple, ordered = false, true
		}
		if box, ok := child.(*BoxElement); ok && box.st.Position.Mode() == properties.PositionAbsolute {
			simple = false
		}
		if _, ok := child.(OutOfFlow); ok {
			simple = false
		}
	}
	if simple {
		return b.children, nil, nil, false
	}
	flow = make([]Element, 0, len(b.children))
	for _, child := range b.children {
		if _, ok := child.(OutOfFlow); ok {
			outside = append(outside, child)
		} else if box, ok := child.(*BoxElement); ok && box.st.Position.Mode() == properties.PositionAbsolute {
			absolute = append(absolute, child)
		} else {
			flow = append(flow, child)
		}
	}
	return flow, absolute, outside, ordered
}

// OutOfFlow is implemented by elements that paint over the window and
// take no space where they're placed: Dialog, Sheet and Drawer. A box
// lays them out apart from its other children, so its Gap and Justify
// leave them out.
type OutOfFlow interface {
	Element
	OutOfFlow()
}

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
	track ChildTracker
}

// mirror places flow children from the right in a box width wide, for
// right-to-left locales (CSS direction: rtl): a row runs from the right
// and a column aligns its start to the right. Positioned children keep
// their physical Left and Right.
func mirror(items []zPaint, width int) {
	for i := range items {
		it := &items[i]
		if it.positioned {
			continue
		}
		x := width - it.off.X - it.size.X
		it.track.Place(image.Pt(x-it.off.X, 0))
		it.off.X = x
	}
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

// mainConstraint returns the main-axis maximum of c.
func mainConstraint(c layout.Constraints, axis layout.Axis) int {
	if axis == layout.Horizontal {
		return c.Max.X
	}
	return c.Max.Y
}

// minConstraint returns the main-axis minimum of c.
func minConstraint(c layout.Constraints, axis layout.Axis) int {
	if axis == layout.Horizontal {
		return c.Min.X
	}
	return c.Min.Y
}

// insetConstraints shrinks c by the inset on each side. Min shrinks by
// the same total inset as Max (clamped at 0), so an exact constraint
// (Min == Max, from a fixed or filled size) stays exact after the
// inset is removed instead of drifting.
func insetConstraints(c layout.Constraints, i properties.Inset) layout.Constraints {
	c.Min.X -= i.Left + i.Right
	c.Max.X -= i.Left + i.Right
	c.Min.Y -= i.Top + i.Bottom
	c.Max.Y -= i.Top + i.Bottom
	if c.Min.X < 0 {
		c.Min.X = 0
	}
	if c.Min.Y < 0 {
		c.Min.Y = 0
	}
	if c.Max.X < c.Min.X {
		c.Max.X = c.Min.X
	}
	if c.Max.Y < c.Min.Y {
		c.Max.Y = c.Min.Y
	}
	return c
}

// unbounded mirrors the sentinel Gio uses for infinite constraints.
const unbounded = 1e6

// isUnbounded reports whether a maximum constraint means "no limit".
// Margins and padding subtract from the sentinel on the way down, so
// anything above half of it counts.
func isUnbounded(max int) bool {
	return max >= unbounded/2
}
