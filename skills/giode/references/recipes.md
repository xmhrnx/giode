# Recipes

Giode isn't opinionated about how you structure components. An `Element` is anything with
a `Layout(gtx layout.Context) layout.Dimensions` method, and the view is a Go function
that runs every frame, so plain Go - functions, structs, maps, closures - is the
component model. These recipes are the patterns that work well; mix them freely, even in
one screen.

Every snippet compiles against the current release.

| Recipe | Reach for it when |
| --- | --- |
| [Function components](#function-components) | A piece of UI is drawn from its arguments |
| [Props structs](#props-structs) | A component has many optional parameters |
| [Stateful structs](#stateful-structs) | A component remembers something between frames |
| [Fluent builders](#fluent-builders) | You want the built-in components' chained style |
| [Children and slots](#children-and-slots) | A layout wraps content the caller supplies |
| [Keyed state in lists](#keyed-state-in-lists) | Rows come and go and each needs its own state |
| [Shared state](#shared-state) | Several parts of the app read the same value, or a goroutine writes it |
| [Model, update, view](#model-update-view) | You like Elm or Redux: one model, messages, a pure view |
| [Variants](#variants) | One component in a few looks |
| [Responsive components](#responsive-components) | Layout changes with the available width |
| [Memoized subtrees](#memoized-subtrees) | Part of the view is costly to build and rarely changes |
| [Custom drawing](#custom-drawing) | Nothing built-in draws it: charts, waveforms, canvases |
| [Wrapping Gio widgets](#wrapping-gio-widgets) | You want Gio's widget state with Giode's styling |

## Function components

The simplest component is a function returning an `Element`. It's stateless, so it's
cheap to call inside the view every frame; its arguments are its props.

```go
func Card(title, body string) giode.Element {
	return giode.Stack(giode.Styles{
		Gap:          6,
		Padding:      giode.UniformInset(16),
		Background:   "theme.surface",
		BorderRadius: 12,
	},
		giode.Text(title, giode.Styles{FontSize: 18, FontWeight: giode.FontWeightSemiBold}),
		giode.Text(body, giode.Styles{Color: "theme.muted"}),
	)
}
```

```go
giode.Stack(giode.Styles{Gap: 12},
	Card("Fast", "Native GPU rendering, no webview."),
	Card("Portable", "One codebase for six platforms."),
)
```

## Props structs

When a component grows optional parameters, take a struct. Zero values are the defaults,
and `styles.Merge` lets callers override any style without your component listing them.

```go
type BadgeProps struct {
	Label string
	Tone  string // "info", "success", "danger"
	Style giode.Styles
}

func Badge(p BadgeProps) giode.Element {
	if p.Tone == "" {
		p.Tone = "info"
	}
	base := giode.Styles{
		Padding:      giode.SymmetricInset(8, 2),
		BorderRadius: 999,
		Background:   giode.Color("theme." + p.Tone),
		Color:        "#ffffff",
		FontSize:     12,
	}
	st := styles.Merge(base, p.Style)
	return giode.Box(st, giode.Text(p.Label, giode.Styles{Color: st.Color, FontSize: st.FontSize}))
}
```

```go
Badge(BadgeProps{Label: "Beta", Tone: "warning"})
Badge(BadgeProps{Label: "New", Style: giode.Styles{FontSize: 14}})
```

## Stateful structs

A component that remembers something - a count, an open panel, a text field's contents -
is a struct with a `Layout` method. Create it **once**, outside the view function, and lay
it out every frame. Built-in stateful components (`Button`, `Input.Text`, `Tabs`) work the
same way, so they live in its fields.

```go
type Counter struct {
	n   int
	inc *giode.ButtonElement
}

func NewCounter() *Counter {
	c := &Counter{}
	c.inc = giode.Button("+1").OnClick(func() { c.n++ })
	return c
}

func (c *Counter) Layout(gtx layout.Context) layout.Dimensions {
	return giode.Row(giode.Styles{Gap: 12, Align: giode.AlignCenter},
		giode.Text(fmt.Sprintf("%d", c.n)),
		c.inc,
	).Layout(gtx)
}
```

```go
counter := NewCounter() // once
app.Run(func() giode.Element {
	return counter // every frame
})
```

## Fluent builders

The built-in components configure with chained setters returning the component. Do the
same for a familiar API:

```go
type Rating struct {
	max, value int
	onChange   func(int)
	color      giode.Color
}

func NewRating(max int) *Rating       { return &Rating{max: max, color: "theme.warning"} }
func (r *Rating) Value(v int) *Rating { r.value = v; return r }
func (r *Rating) Color(c giode.Color) *Rating {
	r.color = c
	return r
}
func (r *Rating) OnChange(fn func(int)) *Rating { r.onChange = fn; return r }
func (r *Rating) Layout(gtx layout.Context) layout.Dimensions {
	stars := make([]giode.Element, r.max)
	for i := range stars {
		c := giode.Color("theme.border")
		if i < r.value {
			c = r.color
		}
		stars[i] = giode.Text("★", giode.Styles{Color: c, FontSize: 22})
	}
	return giode.Row(giode.Styles{Gap: 2}, stars...).Layout(gtx)
}
```

```go
rating := NewRating(5).Value(4).OnChange(func(v int) { save(v) })
```

## Children and slots

Take children as a variadic `...giode.Element` for the main content, and named
parameters (or struct fields) for slots like a header's actions:

```go
func Panel(title string, actions []giode.Element, body ...giode.Element) giode.Element {
	header := giode.Row(giode.Styles{Align: giode.AlignCenter, Gap: 8},
		append([]giode.Element{
			giode.Box(giode.Styles{FlexGrow: 1}, giode.Text(title, giode.Styles{FontWeight: giode.FontWeightSemiBold})),
		}, actions...)...,
	)
	return giode.Stack(giode.Styles{Gap: 12, Padding: giode.UniformInset(16)},
		append([]giode.Element{header}, body...)...,
	)
}
```

```go
Panel("Files", []giode.Element{uploadButton},
	fileList,
	footer,
)
```

## Keyed state in lists

A list whose rows come and go can't create each row's state once up front. `giode.Keyed`
looks state up by a stable key - like a hook in React - creating it the first time and
dropping it when the key hasn't been laid out for a while:

```go
type Todo struct {
	ID   int
	Text string
	Done bool
}

func TodoRows(todos []*Todo) []giode.Element {
	rows := make([]giode.Element, 0, len(todos))
	for _, t := range todos {
		row := giode.Keyed("todos", t.ID, func() *giode.PressableElement {
			return giode.Pressable(func(s giode.PressState) giode.Element {
				bg := giode.Color("transparent")
				if s.Hovered {
					bg = "theme.surface"
				}
				mark := "○ "
				if t.Done {
					mark = "● "
				}
				return giode.Box(giode.Styles{Padding: giode.SymmetricInset(12, 8), Background: bg, BorderRadius: 8},
					giode.Text(mark+t.Text))
			}).OnClick(func(giode.ClickEvent) { t.Done = !t.Done })
		})
		rows = append(rows, row)
	}
	return rows
}
```

Use a key that identifies the item (an ID), not its position, so state follows the item
when the list reorders. For long lists, put the rows in a `LazyList`, which only builds
the visible ones.

## Shared state

`giode.State[T]` is an observable value that any part of the view reads and anything -
including a goroutine - writes. Writing requests a redraw by itself.

```go
var cart = giode.NewState([]string{})

func CartBadge() giode.Element {
	return Badge(BadgeProps{Label: fmt.Sprintf("%d in cart", len(cart.Get()))})
}

func addToCart(item string) {
	cart.Update(func(items []string) []string { return append(items, item) })
}
```

`Watch` runs a callback on every change, for side effects like saving. Name states with
`giode.NamedState` to see and set them from the [MCP server](mcp.md).

## Model, update, view

For one source of truth: a model value, messages describing what happened, a pure
`update` and a `view` of the model. Side effects (a network call) run in goroutines and
report back with a message.

```go
type Model struct {
	Query   string
	Results []string
	Loading bool
}

type Msg interface{}
type SearchStarted struct{ Query string }
type ResultsArrived struct{ Results []string }

var model = giode.NewState(Model{})

func dispatch(msg Msg) {
	model.Update(func(m Model) Model { return update(m, msg) })
}

func update(m Model, msg Msg) Model {
	switch msg := msg.(type) {
	case SearchStarted:
		m.Query, m.Loading = msg.Query, true
		go func() { dispatch(ResultsArrived{Results: search(msg.Query)}) }()
	case ResultsArrived:
		m.Results, m.Loading = msg.Results, false
	}
	return m
}

func view(m Model) giode.Element {
	if m.Loading {
		return giode.Text("Searching " + m.Query + "…")
	}
	rows := make([]giode.Element, len(m.Results))
	for i, r := range m.Results {
		rows[i] = giode.Text(r)
	}
	return giode.Stack(giode.Styles{Gap: 4}, rows...)
}

func search(q string) []string { return []string{q + " 1", q + " 2"} }
```

```go
app.Run(func() giode.Element { return view(model.Get()) })
```

## Variants

Keep variants as data: a map of `Styles` merged over a base. Theme tokens (`theme.primary`,
`theme.surface`, `theme.danger`) follow light and dark mode and custom palettes.

```go
var buttonVariants = map[string]giode.Styles{
	"primary": {Background: "theme.primary", Color: "theme.onprimary"},
	"danger":  {Background: "theme.danger", Color: "#ffffff"},
	"quiet":   {Background: "transparent", Color: "theme.text", BorderColor: "theme.border", BorderWidth: 1},
}

func ActionButton(label, variant string) *giode.ButtonElement {
	st := styles.Merge(giode.Styles{BorderRadius: 8, Padding: giode.SymmetricInset(14, 8)}, buttonVariants[variant])
	return giode.Button(label, st)
}
```

## Responsive components

`Responsive` picks a view by the width it's given (not the window's), so a component
adapts wherever it's placed. Inside, `giode.Compact()` tells nested components which
breakpoint matched.

```go
func Gallery(items []giode.Element) giode.Element {
	return giode.Responsive(
		giode.Breakpoint{MaxWidth: 600, Name: "compact", View: func() giode.Element {
			return giode.Stack(giode.Styles{Gap: 8}, items...)
		}},
		giode.Breakpoint{View: func() giode.Element {
			return giode.Row(giode.Styles{Gap: 16, Wrap: giode.WrapWrap}, items...)
		}},
	)
}
```

## Memoized subtrees

`Memo` caches what a subtree drew and replays it while its dependencies, the `State`s it
reads and its constraints stay the same. Widgets inside keep working; see
[Rendering performance](performance.md).

```go
func Sidebar(user string, projects []string) giode.Element {
	return giode.Memo("sidebar", func() giode.Element {
		rows := []giode.Element{giode.Text(user, giode.Styles{FontWeight: giode.FontWeightSemiBold})}
		for _, p := range projects {
			rows = append(rows, giode.Text(p))
		}
		return giode.Stack(giode.Styles{Gap: 6, Width: 220}, rows...)
	}, user, len(projects))
}
```

## Custom drawing

`giode.Raw` drops to Gio: draw with paths, clips and paint ops, and return your size.

```go
func Sparkline(values []float32, c color.NRGBA) giode.Element {
	return giode.Raw(func(gtx layout.Context) layout.Dimensions {
		size := image.Pt(gtx.Constraints.Max.X, gtx.Dp(40))
		if len(values) < 2 {
			return layout.Dimensions{Size: size}
		}
		var p clip.Path
		p.Begin(gtx.Ops)
		step := float32(size.X) / float32(len(values)-1)
		for i, v := range values {
			pt := f32.Pt(float32(i)*step, float32(size.Y)*(1-v))
			if i == 0 {
				p.MoveTo(pt)
			} else {
				p.LineTo(pt)
			}
		}
		paint.FillShape(gtx.Ops, c, clip.Stroke{Path: p.End(), Width: float32(gtx.Dp(2))}.Op())
		return layout.Dimensions{Size: size}
	})
}
```

## Wrapping Gio widgets

Gio's widgets (`widget.Clickable`, `widget.Bool`, `widget.Editor`, gestures) hold
interaction state; Giode elements draw. Combine them in a struct:

```go
type Toggle struct {
	on    widget.Bool
	label string
}

func (t *Toggle) Layout(gtx layout.Context) layout.Dimensions {
	return t.on.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		bg := giode.Color("theme.border")
		if t.on.Value {
			bg = "theme.primary"
		}
		return giode.Row(giode.Styles{Gap: 8, Align: giode.AlignCenter},
			giode.Box(giode.Styles{Width: 16, Height: 16, BorderRadius: 4, Background: bg}),
			giode.Text(t.label),
		).Layout(gtx)
	})
}
```

## Choosing

- Start with functions. Promote to a struct when something must survive between frames.
- Keep state as close to where it's used as possible: a field, then `Keyed`, then a
  shared `State`.
- Reach for `Memo` only after measuring; most views are fast enough rebuilt every frame.
- Anything here can call anything else: a function component can hold a stateful one
  passed in, and a struct can return function components from `Layout`.

See [Extending the library](extending.md) for writing components that feel built-in:
input passes inside memos, drag handling in frameless windows, and portals.
