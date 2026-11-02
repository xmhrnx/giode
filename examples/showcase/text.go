package main

import "github.com/xmhrnx/giode"

// textPage holds the hoverable label, created once so its hover state
// persists across frames (Text.OnHover makes it stateful, like any
// other element with hover styles - see docs/elements.md#hover).
type textPage struct {
	hoverLabel giode.Element
}

func NewTextPage() *textPage {
	p := &textPage{}
	p.hoverLabel = giode.Text("Hover me", giode.Styles{Color: "#94a3b8"}).OnHover(giode.Styles{
		Color:          "#3b82f6",
		TextDecoration: giode.TextDecorationUnderline,
		Cursor:         giode.CursorPointer,
	})
	return p
}

func (p *textPage) View() giode.Element {
	return giode.Stack(
		boxStyle,
		giode.Text("This is a Text", textStyle),
		p.hoverLabel,
		giode.H1("This is a H1", textStyle),
		giode.H2("This is a H2", textStyle),
		giode.H3("This is a H3", textStyle),
		giode.H4("This is a H4", textStyle),
		giode.H5("This is a H5", textStyle),
		giode.H6("This is a H6", textStyle),
	)
}
