// Package title provides heading text.
package title

import (
	"gioui.org/unit"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// H1 through H6 render heading text with decreasing sizes. User
// styles override the defaults.
func H1(text string, st ...styles.Styles) elements.Element {
	return heading(text, 32, first(st))
}

func H2(text string, st ...styles.Styles) elements.Element {
	return heading(text, 28, first(st))
}

func H3(text string, st ...styles.Styles) elements.Element {
	return heading(text, 24, first(st))
}

func H4(text string, st ...styles.Styles) elements.Element {
	return heading(text, 20, first(st))
}

func H5(text string, st ...styles.Styles) elements.Element {
	return heading(text, 16, first(st))
}

func H6(text string, st ...styles.Styles) elements.Element {
	return heading(text, 14, first(st))
}

func heading(text string, size unit.Sp, st styles.Styles) elements.Element {
	return elements.Text(text, styles.Merge(styles.Styles{
		FontSize:   size,
		FontWeight: properties.FontWeightBold,
	}, st))
}

func first(st []styles.Styles) styles.Styles {
	if len(st) > 0 {
		return st[0]
	}
	return styles.Styles{}
}
