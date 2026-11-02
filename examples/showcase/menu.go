package main

import (
	"log"

	"github.com/xmhrnx/giode"
	giodemenu "github.com/xmhrnx/giode/components/menu"
)

// NewShowcaseMenu builds the global right-click context menu. Unlike
// the other components it is not placed in the view tree: register it
// once with App.SetContextMenu and the app renders it automatically.
func NewShowcaseMenu() *giodemenu.Menu {
	return giode.Menu().
		IconItem(mustIcon("star"), "Starred action", func() { log.Println("starred") }).
		Item("Plain action", func() { log.Println("clicked") }).
		Separator().
		DisabledItem("Disabled")
}

func MenuHint() giode.Element {
	return giode.Stack(boxStyle,
		giode.Text("Menu", textStyle),
		giode.Text("Right-click anywhere", giode.Styles{Color: "white"}),
	)
}

// dropdownMenuPage demonstrates Menu.Trigger (a click-to-open, widget-
// anchored dropdown menu) and Menu.SubMenu (a hover-triggered nested
// panel), created once so their open/hover state persists.
type dropdownMenuPage struct {
	trigger giode.Element
	menu    giode.Element
}

func NewDropdownMenuPage() *dropdownMenuPage {
	share := giode.Menu().
		Item("Email", func() { log.Println("share: email") }).
		Item("Link", func() { log.Println("share: link") })

	m := giode.Menu().
		Item("New", func() { log.Println("file: new") }).
		SubMenu("Share", share).
		Separator().
		Item("Quit", func() { log.Println("file: quit") })

	trigger := giode.Button("File", giode.Styles{
		Background: "gray", BorderRadius: 6,
	})

	return &dropdownMenuPage{
		trigger: m.Trigger(trigger),
		menu:    m,
	}
}

func (p *dropdownMenuPage) View() giode.Element {
	return giode.Stack(boxStyle,
		giode.Text("Dropdown menu", textStyle),
		p.trigger,
		p.menu,
	)
}

// mustIcon looks up an icon registered in icons.go.
func mustIcon(name string) giode.IconSource {
	src, ok := giode.LookupIcon(name)
	if !ok {
		panic("missing icon " + name)
	}
	return src
}
