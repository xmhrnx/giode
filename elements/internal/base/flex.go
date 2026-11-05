package base

import (
	"gioui.org/layout"

	"github.com/xmhrnx/giode/styles"
)

// FlexHints marks elements that carry layout hints: how much they grow
// on the flex main axis, their percentage size and their z-index.
type FlexHints interface {
	FlexWeight(axis layout.Axis) int
	FlexPercent(axis layout.Axis) int
	ZIndex() int
}

// WeightOf returns an element's flex grow weight for axis.
func WeightOf(st styles.Styles, axis layout.Axis) int {
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

// PercentOf returns an element's percentage size for axis.
func PercentOf(st styles.Styles, axis layout.Axis) int {
	if axis == layout.Horizontal {
		return int(st.WidthPct)
	}
	return int(st.HeightPct)
}

// FlexWeightOf returns el's flex grow weight for axis, or 0.
func FlexWeightOf(el Element, axis layout.Axis) int {
	if f, ok := el.(FlexHints); ok {
		return f.FlexWeight(axis)
	}
	return 0
}

// FlexPercentOf returns el's percentage size for axis, or 0.
func FlexPercentOf(el Element, axis layout.Axis) int {
	if f, ok := el.(FlexHints); ok {
		return f.FlexPercent(axis)
	}
	return 0
}

// ZIndexOf returns el's z-index, or 0.
func ZIndexOf(el Element) int {
	if f, ok := el.(FlexHints); ok {
		return f.ZIndex()
	}
	return 0
}

// Unbounded mirrors the sentinel Gio uses for infinite constraints.
const Unbounded = 1e6

// IsUnbounded reports whether a maximum constraint means "no limit".
// Margins and padding subtract from the sentinel on the way down, so
// anything above half of it counts.
func IsUnbounded(max int) bool {
	return max >= Unbounded/2
}
