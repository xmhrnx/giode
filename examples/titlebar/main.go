// Command titlebar shows a frameless window with Giode's custom title
// bar: drag it to move the window, double-click it to maximize, use
// the File menu, and switch between the Windows and Mac looks.
package main

import (
	"image"
	"log"

	"github.com/xmhrnx/giode"
)

func main() {
	giode.UseGofont()
	giode.SetMode(giode.ModeDark)

	app := giode.New(giode.Config{
		Title:      "Custom title bar",
		Size:       image.Pt(720, 460),
		MinWidth:   420,
		MinHeight:  240,
		Resizable:  true,
		Frameless:  true, // drop the OS frame; the TitleBar replaces it
		Background: "theme.background",
		OnWindowStateChange: func(s giode.WindowState) {
			log.Println("window:", s)
		},
	})

	file := giode.Menu().
		Item("New", func() { log.Println("new") }).
		Item("Open…", func() { log.Println("open") }).
		Separator().
		Item("Quit", app.Close)
	fileBtn := giode.Button("File", giode.Styles{Color: "theme.text", Padding: giode.SymmetricInset(10, 4), FontSize: 13}).
		Variant(giode.VariantGhost)

	logo := giode.IconSVG("M12 2L22 20H2Z", giode.Styles{Color: "theme.primary", Width: 16})
	bar := giode.TitleBar(app, "Custom title bar", giode.Styles{BorderWidth: 1, BorderColor: "theme.border"}).
		Icon(logo).
		Leading(file.Trigger(fileBtn))

	mac := false
	toggleLook := giode.Button("Switch to Mac look").OnClick(func() {
		mac = !mac
		if mac {
			bar.Variant(giode.TitleBarMac)
		} else {
			bar.Variant(giode.TitleBarWindows)
		}
	})
	toggleTheme := giode.Button("Toggle theme").Variant(giode.VariantOutlined).OnClick(giode.ToggleMode)

	if err := app.Run(func() giode.Element {
		label := "Switch to Mac look"
		if mac {
			label = "Switch to Windows look"
		}
		toggleLook.Label(label)
		return giode.Stack(giode.Styles{Width: -1, Height: -1},
			bar,
			giode.Stack(giode.Styles{Width: -1, FlexGrow: 1, Padding: giode.UniformInset(24), Gap: 12},
				giode.Text("Frameless window", giode.Styles{Color: "theme.text", FontSize: 20, FontWeight: giode.FontWeightBold}),
				giode.Text("Drag the title bar to move the window, double-click it to maximize, "+
					"or drag its edges to resize.", giode.Styles{Color: "theme.muted"}),
				giode.Row(giode.Styles{Gap: 8}, toggleLook, toggleTheme),
				file,
			),
		)
	}); err != nil {
		log.Fatal(err)
	}
}
