package text

import (
	"strconv"

	"gioui.org/layout"

	"github.com/xmhrnx/giode/elements/internal/base"
)

func (t *TextElement) FlexWeight(axis layout.Axis) int {
	return base.WeightOf(t.st, axis)
}

func (t *TextElement) FlexPercent(axis layout.Axis) int {
	return base.PercentOf(t.st, axis)
}

func (t *TextElement) ZIndex() int {
	return int(t.st.ZIndex)
}

// InlineText returns the text content, for diagnostics.
func (t *TextElement) InlineText() string {
	return t.str
}

// Describe names the text for diagnostics.
func (t *TextElement) Describe() string {
	return "Text " + strconv.Quote(t.str)
}
