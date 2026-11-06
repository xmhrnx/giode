package text

import (
	"gioui.org/layout"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/styles"
)

// withWidths applies st's MinWidth and MaxWidth (dp) to gtx's
// constraints, as a Box applies them to its own: a maximum caps the
// minimum too, so text in an expanding parent still wraps (or ends in
// an ellipsis) at MaxWidth, and a minimum beats a maximum.
func withWidths(gtx layout.Context, st styles.Styles) layout.Context {
	c := &gtx.Constraints
	if st.MaxWidth > 0 {
		m := gtx.Dp(unit.Dp(st.MaxWidth))
		c.Max.X = min(c.Max.X, m)
		c.Min.X = min(c.Min.X, m)
	}
	if st.MinWidth > 0 {
		if m := gtx.Dp(unit.Dp(st.MinWidth)); c.Min.X < m {
			c.Min.X = m
			c.Max.X = max(c.Max.X, m)
		}
	}
	return gtx
}
