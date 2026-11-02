package main

import "github.com/xmhrnx/giode"

// tooltipPage holds the tooltip, created once so its hover state
// persists across frames.
type tooltipPage struct {
	tip giode.Element
}

func NewTooltipPage() *tooltipPage {
	p := &tooltipPage{}
	trigger := giode.Button("Hover me", giode.Styles{
		Background: "gray", BorderRadius: 8,
	})
	p.tip = giode.Tooltip(trigger, "A helpful hint")
	return p
}

func (p *tooltipPage) View() giode.Element {
	return giode.Stack(boxStyle,
		giode.Text("Tooltip", textStyle),
		p.tip,
	)
}
