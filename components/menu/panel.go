package menu

import (
	"image"

	"gioui.org/layout"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// defaultPanelPadding is the panel's padding in dp when New is given
// none.
const defaultPanelPadding = 6

// panel lays out the menu body and processes item clicks.
func (m *Menu) panel(gtx layout.Context) layout.Dimensions {
	margin := gtx.Dp(unit.Dp(kit.Margin))
	st := styles.Merge(styles.Styles{
		FlexDirection: properties.FlexDirectionColumn,
		Background:    properties.CalcColorReverse(defaultBackground),
		BorderRadius:  8,
		Padding:       properties.UniformInset(defaultPanelPadding),
		Color:         properties.CalcColorReverse(defaultText),
	}, m.styles)
	st.FlexDirection = properties.FlexDirectionColumn

	// Bound the panel to the window and let it hug its content: the
	// incoming gtx is bounded by the window regardless of how deep in
	// the tree Layout was called from - see elements.RegisterPortal's
	// use of elements.SetWindowSize - and is typically exact (Min ==
	// Max, the window size) rather than loose, so the minimum must be
	// cleared or the panel would be forced to fill the window.
	win := gtx.Constraints.Max
	gtx.Constraints.Min = image.Point{}
	gtx.Constraints.Max.X = win.X - 2*margin
	gtx.Constraints.Max.Y = win.Y - 2*margin

	width := m.measureWidth(gtx, st)
	if gtx.Source.Enabled() {
		m.panelLeft = elements.AbsolutePosition().X
	}

	rows := make([]elements.Element, 0, len(m.items))
	for i := range m.items {
		rows = append(rows, m.row(&m.items[i], i, width, st))
	}
	return elements.Box(st, elements.Raw(func(gtx layout.Context) layout.Dimensions {
		return m.layoutRows(gtx, rows, st.Color)
	})).Layout(gtx)
}

// panelPaddingTop is the panel's top padding in dp: the Padding given
// to New, or the default.
func (m *Menu) panelPaddingTop() int {
	if m.styles.Padding != (properties.Inset{}) {
		return m.styles.Padding.Top
	}
	return defaultPanelPadding
}

// measureWidth lays each row out on a scratch context to find the
// widest one, so all rows share the same width. Separators are
// skipped: Divider always fills its ambient constraint rather than
// hugging its (nonexistent) content, so measuring it would report the
// scratch context's width instead of anything meaningful.
func (m *Menu) measureWidth(gtx layout.Context, st styles.Styles) int {
	width := 0
	for i := range m.items {
		if m.items[i].separator {
			continue
		}
		dims := elements.Measure(gtx, m.row(&m.items[i], i, 0, st))
		if dims.Size.X > width {
			width = dims.Size.X
		}
	}
	return width
}
