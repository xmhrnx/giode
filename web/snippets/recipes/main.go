// The code on docs/recipes.md, kept here so it compiles. Each "// Rn"
// comment starts a recipe.

package main

import (
	"fmt"
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/styles"
)

// R1
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

// R2
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

// R3
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

// R4
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

// R5
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

// R6
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

// R7
var cart = giode.NewState([]string{})

func CartBadge() giode.Element {
	return Badge(BadgeProps{Label: fmt.Sprintf("%d in cart", len(cart.Get()))})
}

func addToCart(item string) {
	cart.Update(func(items []string) []string { return append(items, item) })
}

// R8
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

// R9
var buttonVariants = map[string]giode.Styles{
	"primary": {Background: "theme.primary", Color: "theme.onprimary"},
	"danger":  {Background: "theme.danger", Color: "#ffffff"},
	"quiet":   {Background: "transparent", Color: "theme.text", BorderColor: "theme.border", BorderWidth: 1},
}

func ActionButton(label, variant string) *giode.ButtonElement {
	st := styles.Merge(giode.Styles{BorderRadius: 8, Padding: giode.SymmetricInset(14, 8)}, buttonVariants[variant])
	return giode.Button(label, st)
}

// R10
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

// R11
func Sidebar(user string, projects []string) giode.Element {
	return giode.Memo("sidebar", func() giode.Element {
		rows := []giode.Element{giode.Text(user, giode.Styles{FontWeight: giode.FontWeightSemiBold})}
		for _, p := range projects {
			rows = append(rows, giode.Text(p))
		}
		return giode.Stack(giode.Styles{Gap: 6, Width: 220}, rows...)
	}, user, len(projects))
}

// R12
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

// R13
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

func main() {
	_ = Card("a", "b")
	_ = NewCounter()
	_ = NewRating(5).Value(3).OnChange(func(int) {})
	_ = Panel("x", nil)
	_ = TodoRows(nil)
	_ = CartBadge()
	addToCart("x")
	_ = view(model.Get())
	_ = ActionButton("Go", "primary")
	_ = Gallery(nil)
	_ = Sidebar("u", nil)
	_ = Sparkline(nil, color.NRGBA{})
	_ = &Toggle{label: "x"}
}
