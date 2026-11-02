package main

import "github.com/xmhrnx/giode"

// DividerShowcase is stateless, so it can be built fresh every frame
// like Text.
func DividerShowcase() giode.Element {
	return giode.Stack(boxStyle,
		giode.Text("Divider", textStyle),
		giode.Text("Above", giode.Styles{Color: "white"}),
		giode.Box(giode.Styles{Width: 120}, giode.Divider(giode.Styles{Color: "#475569"})),
		giode.Text("Below", giode.Styles{Color: "white"}),
	)
}
