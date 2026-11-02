package main

import "github.com/xmhrnx/giode"

// accordionPage holds the accordion, created once so its expand/
// collapse state persists across frames.
type accordionPage struct {
	acc giode.Element
}

func NewAccordionPage() *accordionPage {
	acc := giode.Accordion(giode.Styles{Width: 200}).
		Item("Section one", func() giode.Element {
			return giode.Text("Content for section one.", giode.Styles{Color: "#94a3b8", FontSize: 12})
		}).
		Item("Section two", func() giode.Element {
			return giode.Text("Content for section two.", giode.Styles{Color: "#94a3b8", FontSize: 12})
		}).
		Item("Section three", func() giode.Element {
			return giode.Text("Content for section three.", giode.Styles{Color: "#94a3b8", FontSize: 12})
		})
	return &accordionPage{acc: acc}
}

func (p *accordionPage) View() giode.Element {
	return giode.Stack(boxStyle,
		giode.Text("Accordion", textStyle),
		p.acc,
	)
}
