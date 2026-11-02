package main

import "github.com/xmhrnx/giode"

// tabsPage holds the tab bar, created once so the selected tab
// persists across frames.
type tabsPage struct {
	tabs giode.Element
}

func NewTabsPage() *tabsPage {
	p := &tabsPage{}
	p.tabs = giode.Tabs(giode.Styles{Color: "#3b82f6", BorderColor: "white"},
		giode.Tab{Label: "One", View: func() giode.Element {
			return giode.Text("Tab one content", giode.Styles{Color: "white"})
		}},
		giode.Tab{Label: "Two", View: func() giode.Element {
			return giode.Text("Tab two content", giode.Styles{Color: "white"})
		}},
	)
	return p
}

func (p *tabsPage) View() giode.Element {
	return giode.Stack(boxStyle,
		giode.Text("Tabs", textStyle),
		giode.Box(giode.Styles{Width: 140}, p.tabs),
	)
}
