package button

import (
	"gioui.org/io/pointer"
	"gioui.org/layout"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/components/spinner"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Unstyled buttons follow the theme: a primary fill with on-primary
// text, or primary text and border for the transparent variants.
var (
	defaultBackground = "theme.primary"
	defaultLabel      = "theme.onprimary"
)

// layoutVisual renders the button face inside the clickable region.
func (b *Button) layoutVisual(gtx layout.Context) layout.Dimensions {
	st := styles.Merge(styles.Styles{
		Padding: properties.SymmetricInset(16, 8),
		Justify: properties.JustifyCenter,
		Align:   properties.AlignCenter,
	}, b.st)

	accent := st.Background
	if accent == "" {
		accent = defaultBackground
	}
	hovered := b.clickable.Hovered() && !b.inert()
	pressed := b.clickable.Pressed() && !b.inert()
	shade := func(c string) string { return c }
	switch {
	case pressed:
		shade = kit.Pressed
	case hovered:
		shade = kit.Hovered
	}
	if hovered {
		pointer.CursorPointer.Add(gtx.Ops)
	}

	switch b.variant {
	case VariantOutlined:
		border := st.BorderColor
		if border == "" {
			border = accent
		}
		st.BorderColor = border
		if st.BorderWidth == 0 {
			st.BorderWidth = 1
		}
		st.Background, st.BackgroundGradient = "", nil
		switch {
		case pressed:
			st.Background = kit.Alpha(border, 0.24)
		case hovered:
			st.Background = kit.Alpha(border, 0.12)
		}
	case VariantGhost:
		// Fully transparent at rest: only the label is visible until
		// hover reveals the box.
		st.BorderWidth = 0
		st.Background, st.BackgroundGradient = "", nil
		switch {
		case pressed:
			st.Background = kit.Alpha(accent, 0.24)
		case hovered:
			st.Background = kit.Alpha(accent, 0.12)
		}
	default: // VariantContained, and the zero value for compatibility.
		if g := st.BackgroundGradient; g != nil {
			// Hover and press shade every stop of the gradient.
			st.BackgroundGradient = shadeGradient(g, shade)
			st.Background = ""
		} else {
			accent = shade(accent)
			st.Background = accent
		}
	}

	labelColor := st.Color
	if labelColor == "" {
		labelColor = defaultLabel
		if b.variant == VariantOutlined || b.variant == VariantGhost {
			labelColor = accent
		}
	}
	if b.disabled {
		st.Opacity = 0.5
	}
	// Icons inherit the label color through the box's Color.
	st.Color = labelColor
	st.FlexDirection = properties.FlexDirectionRow
	if st.Gap == 0 {
		st.Gap = 8
	}

	var children []elements.Element
	switch {
	case b.loading:
		children = append(children, spinner.Spinner(styles.Styles{Color: labelColor, Width: 14}))
	case b.icon != nil:
		children = append(children, b.icon)
	}
	if b.label != "" {
		children = append(children, elements.Text(b.label, styles.Styles{
			Color:     labelColor,
			FontSize:  st.FontSize,
			TextAlign: st.TextAlign,
		}))
	}
	if b.trailing != nil {
		children = append(children, b.trailing)
	}
	return elements.Box(st, children...).Layout(gtx)
}

// shadeGradient returns g with fn applied to every stop's color.
func shadeGradient(g *properties.LinearGradient, fn func(string) string) *properties.LinearGradient {
	out := &properties.LinearGradient{Angle: g.Angle, Stops: make([]properties.ColorStop, len(g.Stops))}
	for i, s := range g.Stops {
		out.Stops[i] = properties.ColorStop{Offset: s.Offset, Color: fn(s.Color)}
	}
	return out
}
