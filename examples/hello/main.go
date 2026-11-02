// Command hello is a minimal Giode demo: a frameless, draggable
// window with a counter button and a global right-click menu.
package main

import (
	"fmt"
	"image"
	"log"

	"github.com/xmhrnx/giode"
)

func main() {
	giode.UseGofont()

	count := 0

	btn := giode.Button("Click me", giode.Styles{
		Background:   "#0f172a",
		Color:        "#ffffff",
		Padding:      giode.SymmetricInset(16, 8),
		BorderRadius: 8,
	}).OnClick(func() { count++ })

	menu := giode.Menu().
		Item("Reset counter", func() { count = 0 }).
		Separator().
		Item("Quit", func() { log.Println("bye") })

	app := giode.New(giode.Config{
		Title:      "Giode",
		Size:       image.Pt(800, 600),
		Resizable:  true,
		Frameless:  true,
		Draggable:  true,
		Background: "#0f172a",
	})
	app.SetContextMenu(menu)

	if err := app.Run(func() giode.Element {
		return giode.Box(
			giode.Styles{
				Align:      giode.AlignCenter,
				Justify:    giode.JustifyCenter,
				Width:      -1,
				Height:     -1,
				Background: "black",
			},
			giode.Box(
				giode.Styles{
					FlexDirection: giode.FlexDirectionColumn,
					Align:         giode.AlignCenter,
					Gap:           12,
					Padding:       giode.UniformInset(24),
					Background:    "#1e293b",
					BorderRadius:  12,
					BoxShadow:     giode.BoxShadow{Y: 4, Blur: 12, Color: "#00000060"},
				},
				giode.Text("Welcome to Giode", giode.Styles{Color: "#f8fafc", FontSize: 24}),
				giode.Text(fmt.Sprintf("Clicks: %d", count), giode.Styles{Color: "#94a3b8"}),
				btn,
			),
		)
	}); err != nil {
		log.Fatal(err)
	}
}
