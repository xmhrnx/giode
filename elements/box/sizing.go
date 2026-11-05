package box

import (
	"gioui.org/layout"

	"github.com/xmhrnx/giode/elements/internal/base"
	"github.com/xmhrnx/giode/styles"
)

// applyMinMax clamps the resolved constraints with the min/max size
// properties. A maximum caps the minimum too: an expanding box (Width
// -1, or one a flex parent grows) is handed a minimum equal to the
// space it fills, which would otherwise override MaxWidth - the CSS
// "width: 100%; max-width: 760px" column. The parent's Justify or
// Align then places the narrower box. As in CSS, a minimum beats a
// maximum.
func applyMinMax(c *layout.Constraints, st styles.Styles) {
	if st.MaxWidth > 0 {
		c.Max.X = min(c.Max.X, int(st.MaxWidth))
		c.Min.X = min(c.Min.X, int(st.MaxWidth))
	}
	if st.MaxHeight > 0 {
		c.Max.Y = min(c.Max.Y, int(st.MaxHeight))
		c.Min.Y = min(c.Min.Y, int(st.MaxHeight))
	}
	if st.MinWidth > 0 && c.Min.X < int(st.MinWidth) {
		c.Min.X = int(st.MinWidth)
	}
	if st.MinHeight > 0 && c.Min.Y < int(st.MinHeight) {
		c.Min.Y = int(st.MinHeight)
	}
	if c.Max.X < c.Min.X {
		c.Max.X = c.Min.X
	}
	if c.Max.Y < c.Min.Y {
		c.Max.Y = c.Min.Y
	}
}

// applyPct sizes the axis to pct percent of the available space.
func applyPct(c *layout.Constraints, axis layout.Axis, pct int) {
	if pct <= 0 {
		return
	}
	if pct > 100 {
		pct = 100
	}
	if axis == layout.Horizontal {
		if base.IsUnbounded(c.Max.X) {
			return
		}
		v := c.Max.X * pct / 100
		c.Min.X, c.Max.X = v, v
	} else {
		if base.IsUnbounded(c.Max.Y) {
			return
		}
		v := c.Max.Y * pct / 100
		c.Min.Y, c.Max.Y = v, v
	}
}

// applyAspect derives the hugging axis from the resized axis when an
// aspect ratio is set.
func applyAspect(c *layout.Constraints, st styles.Styles) {
	if st.AspectRatio <= 0 {
		return
	}
	ratio := float32(st.AspectRatio)
	if st.Width != 0 && st.Height == 0 && c.Max.X > 0 && !base.IsUnbounded(c.Max.X) {
		h := int(float32(c.Max.X) / ratio)
		c.Min.Y, c.Max.Y = h, h
	} else if st.Height != 0 && st.Width == 0 && c.Max.Y > 0 && !base.IsUnbounded(c.Max.Y) {
		w := int(float32(c.Max.Y) * ratio)
		c.Min.X, c.Max.X = w, w
	}
}
