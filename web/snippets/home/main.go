// The code on the website's home page, kept here so it compiles. Each
// "// NAME" comment starts a sample the page shows (see vite.config.js).

package main

import (
	"fmt"

	"github.com/xmhrnx/giode"
)

// HERO
func main() {
	count := giode.NewState(0)
	add := giode.Button("Add one", giode.Styles{
		Background:   "theme.primary",
		BorderRadius: 10,
	}).OnClick(func() {
		count.Update(func(n int) int { return n + 1 })
	})

	app := giode.New(giode.Config{Title: "Counter"})
	app.Run(func() giode.Element {
		return giode.Stack(giode.Styles{Gap: 16, Align: giode.AlignCenter},
			giode.Text(fmt.Sprintf("%d", count.Get()), giode.Styles{FontSize: 48}),
			add,
		)
	})
}

// STYLE
func card() giode.Element {
	return giode.Box(giode.Styles{
		FlexDirection: giode.FlexDirectionColumn,
		Gap:           8,
		Padding:       giode.UniformInset(20),
		Background:    "theme.surface",
		BorderRadius:  16,
		BoxShadow:     giode.BoxShadow{Y: 8, Blur: 24, Color: "#0000002a"},
	},
		giode.Text("Weekly report", giode.Styles{FontSize: 18, FontWeight: giode.FontWeightSemiBold}),
		giode.Text("12 tasks done, 3 to go", giode.Styles{Color: "theme.muted"}),
	).OnHover(giode.Styles{BorderColor: "theme.primary", BorderWidth: 1})
}

// STATE
var user = giode.NewState("")

func login() {
	go func() {
		name := fetchUser() // network, disk, anything
		user.Set(name)      // the window redraws by itself
	}()
}

func header() giode.Element {
	return giode.Text("Hello, " + user.Get())
}

// COMPONENTS
func form() giode.Element {
	name := giode.Input.Text("Your name")
	toasts := giode.Toaster()
	save := giode.Button("Save").OnClick(func() {
		toasts.Show(giode.ToastSuccess, "Saved "+name.Text())
	})
	return giode.Stack(giode.Styles{Gap: 12}, name, save, toasts)
}

// RESPONSIVE
func page(sidebar, content giode.Element) giode.Element {
	return giode.Responsive(
		giode.Breakpoint{MaxWidth: 640, Name: "compact", View: func() giode.Element {
			return content // phones: content only
		}},
		giode.Breakpoint{View: func() giode.Element {
			return giode.Row(giode.Styles{Gap: 24}, sidebar, content)
		}},
	)
}

func fetchUser() string { return "" }

var _ = card
var _ = login
var _ = header
var _ = form
var _ = page
