package main

import "github.com/xmhrnx/giode"

// appShellPage holds the shell, created once so its Navbar's Section
// (a Scroll under the hood) keeps its offset across frames instead of
// resetting every frame like a freshly-built stateless element would.
type appShellPage struct {
	shell giode.Element
}

func NewAppShellPage() *appShellPage {
	navbar := giode.Section(func() giode.Element {
		return giode.Stack(giode.Styles{Padding: giode.UniformInset(6), Gap: 4},
			giode.Text("Nav 1", giode.Styles{Color: "white", FontSize: 10}),
			giode.Text("Nav 2", giode.Styles{Color: "white", FontSize: 10}),
			giode.Text("Nav 3", giode.Styles{Color: "white", FontSize: 10}),
			giode.Text("Nav 4", giode.Styles{Color: "white", FontSize: 10}),
			giode.Text("Nav 5", giode.Styles{Color: "white", FontSize: 10}),
		)
	})

	shell := giode.AppShell(giode.AppShellConfig{
		Header: giode.Header(28, giode.Box(
			giode.Styles{Justify: giode.JustifyCenter, Align: giode.AlignCenter, Width: -1, Height: -1},
			giode.Text("Header", giode.Styles{Color: "white", FontSize: 11}),
		)),
		Navbar: giode.Navbar(60, navbar),
		Footer: giode.Footer(22, giode.Box(
			giode.Styles{Justify: giode.JustifyCenter, Align: giode.AlignCenter, Width: -1, Height: -1},
			giode.Text("Footer", giode.Styles{Color: "white", FontSize: 10}),
		)),
		Main: giode.Main(giode.Box(
			giode.Styles{Justify: giode.JustifyCenter, Align: giode.AlignCenter, Width: -1, Height: -1, Background: "#0f172a"},
			giode.Text("Main", giode.Styles{Color: "white", FontSize: 12}),
		)),
	})

	return &appShellPage{shell: shell}
}

// View renders the shell inside a fixed-size preview box: a full-
// scale AppShell fills whatever space it's given (normally the whole
// window), so it's boxed here just to keep it in scale with the rest
// of the gallery.
func (p *appShellPage) View() giode.Element {
	return giode.Stack(boxStyle,
		giode.Text("AppShell", textStyle),
		giode.Box(giode.Styles{Width: 220, Height: 160, BorderRadius: 6, Overflow: giode.OverflowHidden}, p.shell),
	)
}
