package text

import (
	"gioui.org/layout"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/styles"
)

// withWidths applies st's Width, MinWidth and MaxWidth (dp) to gtx's
// constraints, as a Box applies them to its own: a fixed Width wraps
// the text at it and takes that much space, whatever the parent row
// offers more of; -1 fills the space; a maximum caps the minimum too,
// so text in an expanding parent still wraps (or ends in an ellipsis)
// at MaxWidth, and a minimum beats a maximum.
func withWidths(gtx layout.Context, st styles.Styles) layout.Context {
	c := &gtx.Constraints
	switch {
	case st.WidthPct > 0 && c.Max.X < 1e6:
		w := int(float32(c.Max.X) * float32(st.WidthPct) / 100)
		c.Min.X, c.Max.X = w, w
	case st.Width > 0:
		w := min(gtx.Dp(unit.Dp(st.Width)), c.Max.X)
		c.Min.X, c.Max.X = w, w
	case st.Width < 0 && c.Max.X < 1e6:
		c.Min.X = c.Max.X
	}
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
