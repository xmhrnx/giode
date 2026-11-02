package main

import (
	"fmt"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/components/scroll"
)

// scrollObserverPage demonstrates Scroll's intersection-observer-like
// accessors: OnReachBottom loads another batch of items, and each
// item's own background reflects Scroll.Visible - both created once
// so the loaded count and scroll offset persist across frames.
type scrollObserverPage struct {
	loaded int
	scroll *scroll.Scroll
}

const itemHeight = 28

func NewScrollObserverPage() *scrollObserverPage {
	p := &scrollObserverPage{loaded: 8}
	p.scroll = giode.Scroll(func() giode.Element {
		items := make([]giode.Element, 0, p.loaded)
		for i := 1; i <= p.loaded; i++ {
			top := (i - 1) * itemHeight
			bg := "#1e293b"
			if p.scroll.Visible(top, itemHeight) {
				bg = "#334155" // currently within the viewport
			}
			items = append(items, giode.Box(
				giode.Styles{Height: itemHeight, Padding: giode.SymmetricInset(8, 4), Background: bg},
				giode.Text(fmt.Sprintf("Item %d", i), giode.Styles{Color: "white", FontSize: 12}),
			))
		}
		return giode.Stack(giode.Styles{}, items...)
	}, giode.Styles{
		ScrollBar: giode.ScrollBar{Width: 6, ThumbColor: "#3b82f6", TrackColor: "#1e293b"},
	})
	p.scroll.OnReachBottom(func() {
		if p.loaded < 40 {
			p.loaded += 8
		}
	})
	return p
}

func (p *scrollObserverPage) View() giode.Element {
	return giode.Stack(boxStyle,
		giode.Text("Scroll observer", textStyle),
		giode.Text(fmt.Sprintf("%d loaded, %.0f%% scrolled", p.loaded, p.scroll.Progress()*100), giode.Styles{Color: "white", FontSize: 11}),
		giode.Box(giode.Styles{Width: 160, Height: 100}, p.scroll),
	)
}
