package main

import "github.com/xmhrnx/giode"

// themePage demonstrates "theme.*" color tokens: everything below
// follows giode.SetMode without its own styles changing.
type themePage struct {
	dark   bool
	toggle giode.Element
}

func NewThemePage() *themePage {
	p := &themePage{}
	p.toggle = giode.Button("Toggle theme", giode.Styles{
		Background: "theme.primary", Color: "white", BorderRadius: 8,
	}).OnClick(func() {
		p.dark = !p.dark
		if p.dark {
			giode.SetMode(giode.ModeDark)
		} else {
			giode.SetMode(giode.ModeLight)
		}
	})
	return p
}

func (p *themePage) View() giode.Element {
	return giode.Stack(
		giode.Styles{
			Align:        giode.AlignCenter,
			Justify:      giode.JustifyCenter,
			Padding:      giode.SymmetricInset(10, 10),
			Background:   "theme.surface",
			BorderWidth:  1,
			BorderColor:  "theme.border",
			BorderRadius: 5,
			Gap:          10,
			Width:        160,
		},
		giode.Text("Theme", giode.Styles{Color: "theme.text", FontSize: 20}),
		giode.Text("theme.* tokens follow SetMode", giode.Styles{
			Color: "theme.secondary", FontSize: 12, TextAlign: giode.TextAlignCenter,
		}),
		p.toggle,
	)
}
