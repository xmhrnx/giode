package main

import (
	"fmt"

	"github.com/xmhrnx/giode"
)

// buttonPage holds the buttons and the counter, created once so the
// click handler, count and hover/press state persist across frames.
type buttonPage struct {
	count int
	btn   giode.Element

	contained giode.Element
	outlined  giode.Element
	ghost     giode.Element
}

func NewButtonPage() *buttonPage {
	p := &buttonPage{}
	p.btn = giode.Button("Button", giode.Styles{
		Width: 50, Background: "gray", BorderRadius: 10,
		BoxShadow: giode.BoxShadow{X: 1, Y: 1, Blur: 4, Color: "black"},
	}).OnClick(func() { p.count++ })
	p.contained = giode.Button("Contained", giode.Styles{
		Background: "#3b82f6", Color: "white", BorderRadius: 6,
	})
	p.outlined = giode.Button("Outlined", giode.Styles{
		BorderColor: "#3b82f6", Color: "#3b82f6", BorderRadius: 6,
	}).Variant(giode.VariantOutlined)
	p.ghost = giode.Button("Ghost", giode.Styles{
		Background: "#3b82f6", Color: "white", BorderRadius: 6,
	}).Variant(giode.VariantGhost)
	return p
}

func (p *buttonPage) View() giode.Element {
	return giode.Stack(
		boxStyle,
		giode.Text(fmt.Sprintf("Count: %d", p.count), textStyle),
		p.btn,
		giode.Row(giode.Styles{Gap: 8}, p.contained, p.outlined, p.ghost),
	)
}
