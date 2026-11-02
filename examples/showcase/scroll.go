package main

import (
	"fmt"

	"github.com/xmhrnx/giode"
)

// scrollPage holds the scroll container, created once so its offset
// persists across frames. Its content is a wrapping Row of tags —
// Wrap is on by default for Row, so narrowing the box below wraps
// them onto more lines, and Scroll picks up the vertical overflow
// with a fully customized scrollbar.
type scrollPage struct {
	scroll giode.Element
}

func NewScrollPage() *scrollPage {
	p := &scrollPage{}
	p.scroll = giode.Scroll(func() giode.Element {
		tags := make([]giode.Element, 0, 20)
		for i := 1; i <= 20; i++ {
			tags = append(tags, giode.Box(giode.Styles{
				Padding: giode.SymmetricInset(8, 4), Background: "#334155", BorderRadius: 6,
			}, giode.Text(fmt.Sprintf("tag %d", i), giode.Styles{Color: "white", FontSize: 12})))
		}
		return giode.Row(giode.Styles{Gap: 6}, tags...)
	}, giode.Styles{
		ScrollBar: giode.ScrollBar{Width: 6, ThumbColor: "#3b82f6", TrackColor: "#1e293b"},
	})
	return p
}

func (p *scrollPage) View() giode.Element {
	return giode.Stack(boxStyle,
		giode.Text("Scroll + Wrap", textStyle),
		giode.Box(giode.Styles{Width: 160, Height: 100}, p.scroll),
	)
}
