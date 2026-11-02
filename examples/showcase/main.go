package main

import (
	"image"
	"log"

	"github.com/xmhrnx/giode"
)

var boxStyle = giode.Styles{
	Align:        giode.AlignEnd,
	Justify:      giode.JustifyCenter,
	Padding:      giode.SymmetricInset(10, 10),
	BoxShadow:    giode.BoxShadow{X: 1, Y: 1, Blur: 4, Color: "black"},
	Background:   "black",
	BorderRadius: 5,
	Gap:          10,
}
var textStyle = giode.Styles{Color: "white", FontSize: 24, TextAlign: giode.TextAlignCenter}

func main() {
	giode.UseGofont()
	// The gallery has a dark background; theme tokens follow it.
	giode.SetMode(giode.ModeDark)

	app := giode.New(giode.Config{
		Title:     "GIODE SHOWCASE",
		Size:      image.Pt(800, 600),
		Resizable: true,
		Frameless: false,
		Draggable: true,
		Logger:    log.Default(),
		LogLevel:  giode.LogLevelDebug,
		OnStartup: func() { log.Println("lifecycle: startup") },
		OnReady:   func() { log.Println("lifecycle: ready (first frame)") },
		OnShutdown: func() {
			log.Println("lifecycle: shutdown")
		},
		OnBeforeClose: func() { log.Println("lifecycle: close requested") },
		SingleInstanceLock: &giode.SingleInstanceLock{
			UniqueID: "giode-showcase",
			OnSecondInstanceLaunch: func() {
				log.Println("single instance: already running")
			},
		},
	})

	textPg := NewTextPage()
	buttonPg := NewButtonPage()
	checkboxPg := NewCheckboxPage()
	inputPg := NewInputPage()
	sliderPg := NewSliderPage()
	progressPg := NewProgressPage(sliderPg) // reads sliderPg's live value
	dialogPg := NewDialogShowcase()
	dropdownPg := NewDropdownPage()
	drawerPg := NewDrawerPage()
	tabsPg := NewTabsPage()
	scrollPg := NewScrollPage()
	routerPg := NewRouterPage()
	themePg := NewThemePage()
	tooltipPg := NewTooltipPage()
	dropdownMenuPg := NewDropdownMenuPage()
	autocompletePg := NewAutocompletePage()
	appShellPg := NewAppShellPage()
	scrollObserverPg := NewScrollObserverPage()
	statePg := NewStatePage()
	accordionPg := NewAccordionPage()
	paginationPg := NewPaginationPage()

	app.SetContextMenu(NewShowcaseMenu())

	gallery := giode.Scroll(func() giode.Element {
		return giode.Row(giode.Styles{
			Width:   -1,
			Gap:     20,
			Padding: giode.SymmetricInset(10, 10),
		},
			textPg.View(),
			buttonPg.View(),
			checkboxPg.View(),
			inputPg.View(),
			sliderPg.View(),
			progressPg.View(),
			dialogPg.View(),
			dropdownPg.View(),
			drawerPg.View(),
			tabsPg.View(),
			scrollPg.View(),
			SpinnerShowcase(),
			ImageShowcase(),
			IconsShowcase(),
			DividerShowcase(),
			TableShowcase(),
			ResponsiveShowcase(),
			routerPg.View(),
			themePg.View(),
			tooltipPg.View(),
			dropdownMenuPg.View(),
			autocompletePg.View(),
			appShellPg.View(),
			scrollObserverPg.View(),
			statePg.View(),
			accordionPg.View(),
			paginationPg.View(),
			MenuHint(),
		)
	}, giode.Styles{ScrollBar: giode.ScrollBar{ThumbColor: "#3b82f6"}})

	if err := app.Run(func() giode.Element {
		return giode.Stack(giode.Styles{Width: -1, Height: -1},
			giode.Box(giode.Styles{Width: -1, Height: -1, Background: "#0f172a"}, gallery),
			drawerPg.Overlay(), // last, so overlays paint above the rest
			dialogPg.Overlay(),
		)
	}); err != nil {
		log.Fatal(err)
	}
}
