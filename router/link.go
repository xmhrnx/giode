package router

import (
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

var defaultLinkColor = "#60a5fa"

// Link returns a clickable element that navigates to path when
// clicked.
func (r *Router) Link(path, text string, st ...styles.Styles) elements.Element {
	key := path + "\x00" + text
	click, ok := r.linkClicks[key]
	if !ok {
		click = new(widget.Clickable)
		r.linkClicks[key] = click
	}
	var s styles.Styles
	if len(st) > 0 {
		s = st[0]
	}
	return elements.Raw(func(gtx layout.Context) layout.Dimensions {
		if click.Clicked(gtx) {
			r.Navigate(path)
		}
		return click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			col := s.Color
			if col == "" {
				col = defaultLinkColor
			}
			deco := s.TextDecoration
			if click.Hovered() {
				col = kit.Hovered(col)
				deco = properties.TextDecorationUnderline
				pointer.CursorPointer.Add(gtx.Ops)
			}
			return elements.Box(styles.Styles{
				Color:          col,
				FontSize:       s.FontSize,
				FontWeight:     properties.FontWeightMedium,
				TextDecoration: deco,
			}, elements.Text(text)).Layout(gtx)
		})
	})
}
