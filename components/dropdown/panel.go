package dropdown

import (
	"image"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/unit"

	"github.com/mehran9675/giode/components/kit"
	"github.com/mehran9675/giode/components/scroll"
	"github.com/mehran9675/giode/elements"
	"github.com/mehran9675/giode/styles"
	"github.com/mehran9675/giode/styles/properties"
)

// dropdown lays out the option list, at most maxH tall and scrolling
// when the options need more, and processes item clicks.
func (s *Select) dropdown(gtx layout.Context, st styles.Styles, width, maxH int) layout.Dimensions {
	gtx.Constraints.Min = image.Pt(width, 0)
	gtx.Constraints.Max.X = width
	gtx.Constraints.Max.Y = min(gtx.Constraints.Max.Y, maxH)
	st.FlexDirection = properties.FlexDirectionColumn
	st.Padding = properties.UniformInset(4)

	options := func() elements.Element {
		rows := make([]elements.Element, 0, len(s.options))
		for i := range s.options {
			rows = append(rows, s.optionRow(i, st))
		}
		return elements.Box(styles.Styles{FlexDirection: properties.FlexDirectionColumn, Width: -1}, rows...)
	}
	// The column box hands its child an unbounded height, so the
	// room left inside the padding and border is set here.
	inner := gtx.Constraints.Max.Y - gtx.Dp(8) - 2*gtx.Dp(unit.Dp(st.BorderWidth))
	content := elements.Raw(func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.Y = max(min(gtx.Constraints.Max.Y, inner), 0)
		gtx.Constraints.Min.Y = 0
		// Hug the options when they fit; scroll when they don't.
		mgtx := gtx
		mgtx.Constraints.Max.Y = 1 << 24
		if elements.Measure(mgtx, options()).Size.Y <= gtx.Constraints.Max.Y {
			if gtx.Source.Enabled() {
				s.reveal = false
			}
			return options().Layout(gtx)
		}
		if s.list == nil {
			s.list = scroll.New(options)
		}
		dims := s.list.Layout(gtx)
		if s.reveal && gtx.Source.Enabled() {
			s.revealSelected()
			kit.RequestFrame(gtx)
		}
		return dims
	})
	return elements.Box(st, content).Layout(gtx)
}

// revealSelected scrolls the list so the selected option shows, centred
// where it can be. The rows are all one line, so they share a height.
func (s *Select) revealSelected() {
	content, view := s.list.ContentHeight(), s.list.ViewportHeight()
	if content <= 0 || view <= 0 || len(s.options) == 0 {
		return
	}
	s.reveal = false
	row := content / len(s.options)
	top := s.selected*row - (view-row)/2
	s.list.ScrollTo(float32(max(0, min(top, content-view))))
}

// optionRow renders one hoverable option.
func (s *Select) optionRow(i int, st styles.Styles) elements.Element {
	click := s.itemClick[i]
	return elements.Raw(func(gtx layout.Context) layout.Dimensions {
		if click.Clicked(gtx) {
			s.selected = i
			s.open = false
			if s.onChange != nil {
				s.onChange(i)
			}
			kit.RequestFrame(gtx)
		}
		dims := click.Layout(gtx, kit.Unselectable(func(gtx layout.Context) layout.Dimensions {
			rowSt := styles.Styles{
				Padding:      properties.SymmetricInset(12, 8),
				BorderRadius: 4,
			}
			if click.Hovered() {
				rowSt.Background = s.hover
				if rowSt.Background == "" {
					rowSt.Background = properties.CalcColorReverse(defaultHover)
				}
				pointer.CursorPointer.Add(gtx.Ops)
			}
			return elements.Box(rowSt, elements.Text(s.options[i], styles.Styles{Color: st.Color})).Layout(gtx)
		}))
		// Keep the option clickable even when a frameless window's
		// whole-window drag area happens to reach this portal - see
		// button.Layout's identical comment.
		kit.BlockDrag(gtx, dims.Size)
		return dims
	})
}
