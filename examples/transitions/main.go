// Command transitions shows every built-in transition kind, with the
// speed and easing chosen from the toolbar.
package main

import (
	"image"
	"log"
	"time"

	"github.com/xmhrnx/giode"
)

type demo struct {
	name string
	kind giode.TransitionKind
	t    *giode.TransitionElement
}

func main() {
	giode.UseGofont()
	giode.SetMode(giode.ModeDark)

	app := giode.New(giode.Config{
		Title:      "Transitions",
		Size:       image.Pt(900, 560),
		Resizable:  true,
		Background: "theme.background",
	})

	speeds := []time.Duration{giode.TransitionFast, giode.TransitionNormal, giode.TransitionSlow, 2 * time.Second}
	easings := []giode.Easing{giode.EaseOut, giode.EaseInOut, giode.EaseIn, giode.EaseLinear, giode.EaseOutBack, giode.EaseOutBounce}
	easingNames := make([]string, len(easings))
	for i, e := range easings {
		easingNames[i] = string(e)
	}

	card := func(label string) giode.Element {
		return giode.Box(giode.Styles{
			Width: 150, Height: 80,
			BackgroundGradient: giode.Linear(135, "#3b82f6", "#8b5cf6"),
			BorderRadius:       12, Justify: giode.JustifyCenter, Align: giode.AlignCenter,
		}, giode.Text(label, giode.Styles{Color: "white", FontWeight: giode.FontWeightBold}))
	}

	demos := []*demo{
		{name: "Fade", kind: giode.Fade},
		{name: "SlideUp", kind: giode.SlideUp},
		{name: "SlideDown", kind: giode.SlideDown},
		{name: "SlideLeft", kind: giode.SlideLeft},
		{name: "SlideRight", kind: giode.SlideRight},
		{name: "Enlarge", kind: giode.Enlarge},
		{name: "Shrink", kind: giode.Shrink},
		{name: "Rotate", kind: giode.Rotate},
		{name: "FadeUp", kind: giode.FadeUp},
		{name: "Zoom", kind: giode.Zoom},
		{name: "Pop", kind: giode.Pop},
		{name: "Collapse", kind: giode.Collapse | giode.Fade},
	}
	for _, d := range demos {
		d.t = giode.Transition(card(d.name), d.kind, giode.TransitionSlow, giode.EaseOut)
	}

	speed := giode.Input.Select(giode.Styles{Width: 120}, "Fast", "Normal", "Slow", "2 seconds")
	speed.SetSelected(2)
	easing := giode.Input.Select(giode.Styles{Width: 150}, easingNames...)
	apply := func() {
		for _, d := range demos {
			d.t.Duration(speeds[speed.Selected()]).Easing(easings[easing.Selected()])
		}
	}
	speed.OnChange(func(int) { apply() })
	easing.OnChange(func(int) { apply() })

	toggle := giode.Button("Hide all").OnClick(func() {
		for _, d := range demos {
			d.t.Toggle()
		}
	})

	if err := app.Run(func() giode.Element {
		label := "Hide all"
		if !demos[0].t.Visible() {
			label = "Show all"
		}
		toggle.Label(label)

		cells := make([]giode.Element, 0, len(demos))
		for _, d := range demos {
			cells = append(cells, giode.Stack(giode.Styles{Gap: 6, Width: 170, Height: 120},
				giode.Text(d.name, giode.Styles{Color: "theme.muted", FontSize: 12}),
				d.t,
			))
		}
		return giode.Stack(giode.Styles{Width: -1, Height: -1, Padding: giode.UniformInset(20), Gap: 16},
			giode.Row(giode.Styles{Gap: 12, Align: giode.AlignCenter, Wrap: giode.WrapNoWrap},
				toggle,
				giode.Text("Speed", giode.Styles{Color: "theme.text"}), speed,
				giode.Text("Easing", giode.Styles{Color: "theme.text"}), easing,
			),
			giode.Row(giode.Styles{Gap: 12, Width: -1}, cells...),
		)
	}); err != nil {
		log.Fatal(err)
	}
}
