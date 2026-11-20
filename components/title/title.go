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
// Screen readers announce them as headings of their level.
func H1(text string, st ...styles.Styles) elements.Element {
	return heading(text, 1, 32, first(st))
}

func H2(text string, st ...styles.Styles) elements.Element {
	return heading(text, 2, 28, first(st))
}

func H3(text string, st ...styles.Styles) elements.Element {
	return heading(text, 3, 24, first(st))
}

func H4(text string, st ...styles.Styles) elements.Element {
	return heading(text, 4, 20, first(st))
}

func H5(text string, st ...styles.Styles) elements.Element {
	return heading(text, 5, 16, first(st))
}

func H6(text string, st ...styles.Styles) elements.Element {
	return heading(text, 6, 14, first(st))
}

func heading(text string, level int, size unit.Sp, st styles.Styles) elements.Element {
	t := elements.Text(text, styles.Merge(styles.Styles{
		FontSize:   size,
		FontWeight: properties.FontWeightBold,
	}, st))
	return elements.Accessible(t, func(n *elements.SemanticNode) {
		if n.Kind == elements.SemanticText {
			n.Kind, n.Level = elements.SemanticHeading, level
		}
	})
}

func first(st []styles.Styles) styles.Styles {
	if len(st) > 0 {
		return st[0]
	}
	return styles.Styles{}
}
