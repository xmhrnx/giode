package main

import (
	"github.com/xmhrnx/giode"
	giorouter "github.com/xmhrnx/giode/router"
)

// routerPage holds the router, created once so the current path
// persists across frames. Kept as the concrete type so View can call
// Link, which Router.Layout alone (the plain Element) doesn't expose.
type routerPage struct {
	router *giorouter.Router
}

func NewRouterPage() *routerPage {
	r := giode.NewRouter(
		giode.Route{Path: "/", View: func() giode.Element {
			return giode.Text("Home", giode.Styles{Color: "white"})
		}},
		giode.Route{Path: "/about", View: func() giode.Element {
			return giode.Text("About", giode.Styles{Color: "white"})
		}},
	)
	return &routerPage{router: r}
}

func (p *routerPage) View() giode.Element {
	return giode.Stack(boxStyle,
		giode.Text("Router", textStyle),
		giode.Row(giode.Styles{Gap: 8},
			p.router.Link("/", "Home"),
			p.router.Link("/about", "About"),
		),
		p.router,
	)
}
