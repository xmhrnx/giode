package main

import "github.com/xmhrnx/giode"

// ResponsiveShowcase is stateless, so it can be built fresh every
// frame like Text. Breakpoints are evaluated against the box's own
// available width (160dp here), not the whole window.
func ResponsiveShowcase() giode.Element {
	return giode.Stack(boxStyle,
		giode.Text("Responsive", textStyle),
		giode.Box(giode.Styles{Width: 160},
			giode.Responsive(
				giode.Breakpoint{MaxWidth: 140, View: func() giode.Element {
					return giode.Text("Narrow (<=140dp)", giode.Styles{Color: "white", FontSize: 12})
				}},
				giode.Breakpoint{View: func() giode.Element {
					return giode.Text("Wide (>140dp)", giode.Styles{Color: "white", FontSize: 12})
				}},
			),
		),
	)
}
