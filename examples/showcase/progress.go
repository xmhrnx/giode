package main

import "github.com/xmhrnx/giode"

// progressPage is stateless: it just reads another page's live value
// every frame, so it needs no state of its own beyond that reference.
type progressPage struct {
	slider *sliderPage
}

func NewProgressPage(slider *sliderPage) *progressPage {
	return &progressPage{slider: slider}
}

func (p *progressPage) View() giode.Element {
	return giode.Stack(boxStyle,
		giode.Text("Progress", textStyle),
		giode.Box(giode.Styles{Width: 120},
			giode.Progress(p.slider.Value(), giode.Styles{Color: "#3b82f6", Background: "#334155"}),
		),
	)
}
