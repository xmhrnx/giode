// Command tour is a small task list: a sidebar, a progress card, a text
// field and checkable rows, styled with CSS-like properties and
// following the light or dark theme. The README's recording is of it.
package main

import (
	"fmt"
	"image"
	"log"
	"time"

	"github.com/xmhrnx/giode"
)

type task struct {
	title, tag string
	done       bool
	check      giode.Element
}

func newTask(title, tag string, done bool) *task {
	t := &task{title: title, tag: tag, done: done}
	c := giode.Input.Checkbox("").OnChange(func(on bool) { t.done = on })
	c.SetChecked(done)
	t.check = c
	return t
}

var views = []string{"Today", "Upcoming", "Done"}

func main() {
	giode.SetMode(giode.ModeDark)

	tasks := []*task{
		newTask("Sketch the onboarding screens", "Design", true),
		newTask("Write the release notes", "Docs", false),
		newTask("Review the sync pull request", "Code", false),
		newTask("Record the demo video", "Launch", false),
	}
	view := 0
	// counts holds each view's count, read by the sidebar rows (created
	// once, with Keyed) on every frame.
	var counts [3]int
	var list giode.Element
	scroll := giode.Scroll(func() giode.Element { return list }, giode.Styles{Width: -1, FlexGrow: 1})

	add := func(text string) {
		if text != "" {
			tasks = append(tasks, newTask(text, "Inbox", false))
		}
	}
	field := giode.Input.Text("Add a task…")
	field.Label("New task").Submit(func(text string) { add(text); field.SetText("") })
	addButton := giode.Button("Add", giode.Styles{BorderRadius: 8}).
		OnClick(func() { add(field.Text()); field.SetText("") })
	darkMode := giode.Input.Switch("Dark mode").OnChange(func(on bool) {
		giode.SetMode(map[bool]giode.Mode{true: giode.ModeDark, false: giode.ModeLight}[on])
	})
	darkMode.SetChecked(true)

	app := giode.New(giode.Config{Title: "Tasks", Size: image.Pt(900, 580), Resizable: true, Background: "theme.background"})
	err := app.Run(func() giode.Element {
		done := 0
		for _, t := range tasks {
			if t.done {
				done++
			}
		}

		// The sidebar: a logo, one row per view with a count, and the
		// theme switch at the bottom.
		counts = [3]int{len(tasks) - done, 0, done}
		var nav []giode.Element
		for i, name := range views {
			nav = append(nav, giode.Keyed("nav", i, func() *giode.PressableElement {
				return giode.Pressable(func(s giode.PressState) giode.Element {
					bg, fg := "", "theme.muted"
					switch {
					case view == i:
						bg, fg = "theme.primary/0.15", "theme.primary"
					case s.Hovered:
						bg = "theme.text/0.06"
					}
					return giode.Row(giode.Styles{Width: -1, Padding: giode.SymmetricInset(12, 9), BorderRadius: 8,
						Background: bg, Align: giode.AlignCenter, Wrap: giode.WrapNoWrap},
						giode.Text(name, giode.Styles{FlexGrow: 1, Color: fg, FontWeight: giode.FontWeightMedium}),
						giode.Text(fmt.Sprint(counts[i]), giode.Styles{Color: "theme.muted", FontSize: 12}),
					)
				}).Label(name).OnClick(func(giode.ClickEvent) { view = i })
			}))
		}
		sidebar := giode.Stack(giode.Styles{Width: 220, Height: -1, Padding: giode.UniformInset(16), Gap: 24,
			Background: "theme.surface", BorderColor: "theme.border"},
			giode.Row(giode.Styles{Gap: 10, Align: giode.AlignCenter},
				giode.Box(giode.Styles{Width: 30, Height: 30, BorderRadius: 9, Align: giode.AlignCenter, Justify: giode.JustifyCenter,
					BackgroundGradient: giode.Linear(135, "#6366f1", "#ec4899")},
					giode.Text("✓", giode.Styles{Color: "#ffffff", FontWeight: giode.FontWeightBold})),
				giode.Text("Tasks", giode.Styles{Color: "theme.text", FontSize: 18, FontWeight: giode.FontWeightBold}),
			),
			giode.Stack(giode.Styles{Gap: 4, Width: -1}, nav...),
			giode.Box(giode.Styles{FlexGrow: 1}),
			darkMode,
		)

		// The progress card, in a gradient.
		pct := float32(done) / float32(max(len(tasks), 1))
		progress := giode.Stack(giode.Styles{Width: -1, Padding: giode.UniformInset(20), Gap: 12, BorderRadius: 14,
			BackgroundGradient: giode.Linear(120, "#6366f1", "#8b5cf6", "#ec4899"),
			BoxShadow:          giode.BoxShadow{Y: 8, Blur: 24, Color: "#6366f155"}},
			giode.Text(fmt.Sprintf("%d of %d done", done, len(tasks)), giode.Styles{Color: "#ffffff", FontSize: 22, FontWeight: giode.FontWeightBold}),
			giode.Progress(pct, giode.Styles{Width: -1, Height: 8, Background: "#ffffff40", Color: "#ffffff"}),
		)

		// The task rows: a checkbox, the title and a tag.
		var rows []giode.Element
		for _, t := range tasks {
			if (view == 1) || (view == 2 && !t.done) {
				continue
			}
			title := giode.Styles{Color: "theme.text", FontSize: 15}
			if t.done {
				title.Color, title.TextDecoration = "theme.muted", giode.TextDecorationLineThrough
			}
			rows = append(rows, giode.Row(giode.Styles{Width: -1, Padding: giode.SymmetricInset(14, 12), Gap: 12,
				Align: giode.AlignCenter, Wrap: giode.WrapNoWrap, BorderRadius: 10, Background: "theme.surface",
				BorderWidth: 1, BorderColor: "theme.border"},
				t.check,
				giode.Box(giode.Styles{FlexGrow: 1}, giode.Text(t.title, title)),
				giode.Chip(t.tag, giode.Styles{Background: "theme.primary/0.12", Color: "theme.primary", BorderColor: "theme.primary/0.12"}),
			))
		}
		if len(rows) == 0 {
			rows = append(rows, giode.Text("Nothing here yet.", giode.Styles{Color: "theme.muted"}))
		}
		list = giode.Stack(giode.Styles{Width: -1, Gap: 8}, rows...)

		main := giode.Stack(giode.Styles{FlexGrow: 1, Height: -1, Padding: giode.SymmetricInset(32, 28), Gap: 20},
			giode.Stack(giode.Styles{Gap: 2},
				giode.Text(views[view], giode.Styles{Color: "theme.text", FontSize: 28, FontWeight: giode.FontWeightBold}),
				giode.Text(time.Now().Format("Monday, January 2"), giode.Styles{Color: "theme.muted"}),
			),
			progress,
			giode.Row(giode.Styles{Width: -1, Gap: 8, Align: giode.AlignCenter, Wrap: giode.WrapNoWrap},
				giode.Box(giode.Styles{FlexGrow: 1}, field), addButton),
			scroll,
		)
		return giode.Row(giode.Styles{Width: -1, Height: -1, Wrap: giode.WrapNoWrap}, sidebar, main)
	})
	if err != nil {
		log.Fatal(err)
	}
}
