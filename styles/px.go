package styles

import (
	"gioui.org/unit"

	"github.com/mehran9675/giode/styles/properties"
)

// Px returns s with every length resolved from dp to physical pixels
// through m. Sentinel values (Width/Height -1, meaning "expand") are
// kept. Elements call it once at the start of Layout; custom elements
// reading lengths from Styles should do the same:
//
//	st := e.st.Px(gtx.Metric)
//	size := int(st.Width)
func (s Styles) Px(m unit.Metric) Styles {
	if m.PxPerDp == 0 || m.PxPerDp == 1 {
		return s
	}
	dp := func(v int) int { return m.Dp(unit.Dp(v)) }
	size := func(v int) int {
		if v <= 0 {
			return v
		}
		return dp(v)
	}
	inset := func(i properties.Inset) properties.Inset {
		// Copied so NoInset stays set.
		o := i
		o.Top, o.Right, o.Bottom, o.Left, o.Start, o.End = dp(i.Top), dp(i.Right), dp(i.Bottom), dp(i.Left), dp(i.Start), dp(i.End)
		return o
	}
	s.Gap = properties.Gap(dp(int(s.Gap)))
	s.Width = properties.Width(size(int(s.Width)))
	s.Height = properties.Height(size(int(s.Height)))
	s.Padding = inset(s.Padding)
	s.Margin = inset(s.Margin)
	s.BorderWidth = properties.BorderWidth(dp(int(s.BorderWidth)))
	s.BorderRadius = properties.BorderRadius(dp(int(s.BorderRadius)))
	c := s.BorderRadii
	s.BorderRadii = properties.Corners{TopLeft: dp(c.TopLeft), TopRight: dp(c.TopRight), BottomRight: dp(c.BottomRight), BottomLeft: dp(c.BottomLeft)}
	// Flush (an explicit 0) stays as it is.
	edge := func(v int) int {
		if v == properties.Flush {
			return v
		}
		return dp(v)
	}
	s.Top = properties.Top(edge(int(s.Top)))
	s.Right = properties.Right(edge(int(s.Right)))
	s.Bottom = properties.Bottom(edge(int(s.Bottom)))
	s.Left = properties.Left(edge(int(s.Left)))
	s.MinWidth = properties.MinWidth(dp(int(s.MinWidth)))
	s.MaxWidth = properties.MaxWidth(dp(int(s.MaxWidth)))
	s.MinHeight = properties.MinHeight(dp(int(s.MinHeight)))
	s.MaxHeight = properties.MaxHeight(dp(int(s.MaxHeight)))
	s.BoxShadow.X = dp(s.BoxShadow.X)
	s.BoxShadow.Y = dp(s.BoxShadow.Y)
	s.BoxShadow.Blur = dp(s.BoxShadow.Blur)
	s.BoxShadow.Spread = dp(s.BoxShadow.Spread)
	s.ScrollBar.Width = dp(s.ScrollBar.Width)
	s.ScrollBar.Radius = dp(s.ScrollBar.Radius)
	s.ScrollBar.MinThumbLength = dp(s.ScrollBar.MinThumbLength)
	s.Transform.TranslateX = dp(s.Transform.TranslateX)
	s.Transform.TranslateY = dp(s.Transform.TranslateY)
	return s
}

// Corners returns the resolved per-corner radii: BorderRadii when set,
// otherwise BorderRadius on every corner.
func (s Styles) Corners() properties.Corners {
	return properties.ResolveCorners(s.BorderRadius, s.BorderRadii)
}

// With returns s with the non-zero values of each of over applied on
// top, in order - the method form of Merge:
//
//	card := base.With(styles.Styles{Background: "theme.surface"})
func (s Styles) With(over ...Styles) Styles {
	return Merge(s, over...)
}
