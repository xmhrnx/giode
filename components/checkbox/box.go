package checkbox

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

var (
	defaultBox     = "#475569"
	defaultChecked = "#3b82f6"
	defaultText    = "#e2e8f0"
	defaultIcon    = "#ffffff"
)

const (
	defaultBoxSize     = 18
	defaultBorderWidth = 2
	defaultRadius      = 4
)

// box renders the square indicator: filled with an icon when checked,
// outlined when not.
func (c *Checkbox) box() elements.Element {
	return elements.Raw(func(gtx layout.Context) layout.Dimensions {
		st := c.st.Px(gtx.Metric)
		size := c.boxSize(gtx)
		gtx.Constraints.Min = size
		gtx.Constraints.Max = size

		radius := int(st.BorderRadius)
		if radius == 0 {
			radius = gtx.Dp(unit.Dp(defaultRadius))
		}
		borderWidth := int(st.BorderWidth)
		if borderWidth == 0 {
			borderWidth = gtx.Dp(unit.Dp(defaultBorderWidth))
		}

		checked := c.bool.Value
		fill := stOrDefault(c.st.Background, defaultChecked)
		border := stOrDefault(c.st.BorderColor, defaultBox)
		switch {
		case !checked && c.bool.Hovered():
			border = kit.Hovered(border)
		case checked && c.bool.Hovered():
			fill = kit.Hovered(fill)
		}

		defer clip.UniformRRect(image.Rectangle{Max: size}, radius).Push(gtx.Ops).Pop()
		if checked {
			paint.Fill(gtx.Ops, properties.CalcColor(fill))
			if c.icon != nil {
				// Icon sizes are in dp: convert the box's pixel size.
				dpSize := properties.Width(float32(size.X) / max(gtx.Metric.PxPerDp, 1))
				iconSt := styles.Merge(styles.Styles{
					Color:  defaultIcon,
					Width:  dpSize,
					Height: properties.Height(dpSize),
				}, c.iconSt)
				gtx.Constraints = layout.Exact(size)
				c.icon.Icon(iconSt).Layout(gtx)
			} else {
				col := properties.ResolveColor(stOrDefault(c.iconSt.Color, defaultIcon))
				gtx.Constraints = layout.Exact(size)
				kit.DrawGlyph(gtx, kit.GlyphCheck, unit.Dp(float32(size.X)/max(gtx.Metric.PxPerDp, 1)), col)
			}
		} else {
			properties.PaintBorder(gtx.Ops, size, properties.BorderWidth(borderWidth), border, properties.BorderRadius(radius))
		}
		return layout.Dimensions{Size: size}
	})
}

// boxSize resolves the indicator size in pixels from Width/Height (in
// dp), defaulting to a square 18x18dp box.
func (c *Checkbox) boxSize(gtx layout.Context) image.Point {
	w := int(c.st.Width)
	h := int(c.st.Height)
	if w <= 0 {
		w = defaultBoxSize
	}
	if h <= 0 {
		h = defaultBoxSize
	}
	return image.Pt(gtx.Dp(unit.Dp(w)), gtx.Dp(unit.Dp(h)))
}

// stOrDefault returns c unless it is unset, in which case it returns
// def.
func stOrDefault(c, def properties.Color) properties.Color {
	if c == "" {
		return def
	}
	return c
}
