package combobox

import (
	"image"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// List is the option panel: a highlighted option moved with the
// keyboard, group headings, and loading, error and empty states. The
// inputs here and the command palette use it; it's exported for others
// like them.
type List struct {
	// Items are the options shown, in order.
	Items []Option
	// Index is the highlighted option.
	Index int
	// Loading shows LoadingText, Err the error and an empty Items
	// EmptyText ("No results" by default).
	Loading                bool
	Err                    error
	LoadingText, EmptyText string
	// MaxRows caps the panel's height in rows: 8 by default.
	MaxRows int
	// Plain leaves out the panel's background and border, for a list
	// inside another panel.
	Plain bool
	// OnPick is called when an option is clicked.
	OnPick func(Option)

	clicks []widget.Clickable
	list   layout.List
	follow bool
}

// SetItems replaces the options, keeping the highlight in range on the
// first enabled one.
func (l *List) SetItems(items []Option) {
	l.Items = items
	l.Index = 0
	l.skipDisabled(1)
	l.follow = true
}

// Move moves the highlight by delta, wrapping, over disabled options.
func (l *List) Move(delta int) {
	n := len(l.Items)
	if n == 0 {
		return
	}
	l.Index = ((l.Index+delta)%n + n) % n
	dir := 1
	if delta < 0 {
		dir = -1
	}
	l.skipDisabled(dir)
	l.follow = true
}

// First and Last move the highlight to the ends.
func (l *List) First() { l.Index = 0; l.skipDisabled(1); l.follow = true }
func (l *List) Last()  { l.Index = len(l.Items) - 1; l.skipDisabled(-1); l.follow = true }

func (l *List) skipDisabled(dir int) {
	for range l.Items {
		if l.Index < 0 || l.Index >= len(l.Items) || !l.Items[l.Index].Disabled {
			return
		}
		l.Index = (l.Index + dir + len(l.Items)) % len(l.Items)
	}
}

// Selected is the highlighted option.
func (l *List) Selected() (Option, bool) {
	if l.Index < 0 || l.Index >= len(l.Items) || l.Items[l.Index].Disabled {
		return Option{}, false
	}
	return l.Items[l.Index], true
}

// rows are the panel's lines: a heading (item -1) or an option.
type row struct {
	heading string
	item    int
}

func (l *List) rows() []row {
	var rows []row
	group := ""
	for i, o := range l.Items {
		if o.Group != "" && o.Group != group {
			rows = append(rows, row{heading: o.Group, item: -1})
		}
		group = o.Group
		rows = append(rows, row{item: i})
	}
	return rows
}

// Layout draws the panel width wide.
func (l *List) Layout(gtx layout.Context, width int) layout.Dimensions {
	for len(l.clicks) < len(l.Items) {
		l.clicks = append(l.clicks, widget.Clickable{})
	}
	gtx.Constraints.Min.X, gtx.Constraints.Max.X = width, width
	gtx.Constraints.Min.Y = 0
	panel := styles.Styles{Background: "theme.surface", BorderRadius: 8, Padding: properties.UniformInset(4),
		FlexDirection: properties.FlexDirectionColumn, BorderWidth: 1, BorderColor: "theme.border", Width: -1}
	if l.Plain {
		panel = styles.Styles{Padding: properties.UniformInset(6), FlexDirection: properties.FlexDirectionColumn, Width: -1}
	}
	var body elements.Element
	switch {
	case l.Loading && len(l.Items) == 0:
		body = l.note(l.LoadingText, "Loading…")
	case l.Err != nil && len(l.Items) == 0:
		body = elements.Box(styles.Styles{Padding: properties.SymmetricInset(10, 8)},
			elements.Text(l.Err.Error(), styles.Styles{Color: "theme.danger"}))
	case len(l.Items) == 0:
		body = l.note(l.EmptyText, "No results")
	default:
		body = elements.Raw(l.layoutRows)
	}
	dims := elements.Box(panel, body).Layout(gtx)
	kit.BlockDrag(gtx, dims.Size)
	return dims
}

func (l *List) note(text, def string) elements.Element {
	if text == "" {
		text = def
	}
	return elements.Box(styles.Styles{Padding: properties.SymmetricInset(10, 8)},
		elements.Text(text, styles.Styles{Color: "theme.muted"}))
}

func (l *List) layoutRows(gtx layout.Context) layout.Dimensions {
	rows := l.rows()
	maxRows := l.MaxRows
	if maxRows <= 0 {
		maxRows = 8
	}
	rowH := gtx.Dp(34)
	gtx.Constraints.Max.Y = min(gtx.Constraints.Max.Y, rowH*maxRows)
	l.list.Axis = layout.Vertical
	if l.follow {
		l.follow = false
		// Keep the highlighted row in view.
		for ri, r := range rows {
			if r.item == l.Index {
				first := l.list.Position.First
				if ri < first+1 {
					l.list.Position = layout.Position{First: max(ri-1, 0)}
				} else if ri >= first+maxRows-1 {
					l.list.Position = layout.Position{First: ri - maxRows + 1}
				}
			}
		}
	}
	return l.list.Layout(gtx, len(rows), func(gtx layout.Context, ri int) layout.Dimensions {
		r := rows[ri]
		if r.item < 0 {
			return elements.Box(styles.Styles{Padding: properties.Inset{Top: 8, Bottom: 4, Left: 10, Right: 10}},
				elements.Text(r.heading, styles.Styles{Color: "theme.muted", FontSize: 12, FontWeight: properties.FontWeightSemiBold})).Layout(gtx)
		}
		return l.layoutOption(gtx, r.item)
	})
}

func (l *List) layoutOption(gtx layout.Context, i int) layout.Dimensions {
	o := l.Items[i]
	click := &l.clicks[i]
	if click.Clicked(gtx) && !o.Disabled {
		l.Index = i
		if l.OnPick != nil {
			l.OnPick(o)
		}
		kit.RequestFrame(gtx)
	}
	return click.Layout(gtx, kit.Unselectable(func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		st := styles.Styles{Padding: properties.SymmetricInset(10, 7), BorderRadius: 4, Gap: 10, Width: -1,
			FlexDirection: properties.FlexDirectionRow, Align: properties.AlignCenter}
		if !o.Disabled && (i == l.Index || click.Hovered()) {
			st.Background = "theme.border"
		}
		if !o.Disabled {
			pointer.CursorPointer.Add(gtx.Ops)
		}
		var kids []elements.Element
		if o.Icon != nil {
			kids = append(kids, o.Icon)
		}
		kids = append(kids, elements.Box(styles.Styles{FlexGrow: 1}, highlighted(o)))
		if o.Detail != "" {
			kids = append(kids, elements.Text(o.Detail, styles.Styles{Color: "theme.muted", FontSize: 12, MaxLines: 1}))
		}
		return elements.Box(st, kids...).Layout(gtx)
	}))
}

// highlighted is the label with the matched letters bold.
func highlighted(o Option) elements.Element {
	color := properties.Color("theme.text")
	if o.Disabled {
		color = "theme.muted"
	}
	text := o.text()
	if len(o.matched) == 0 {
		return elements.Text(text, styles.Styles{Color: color, MaxLines: 1})
	}
	rs := []rune(text)
	var spans []elements.Span
	plain := styles.Styles{Color: color}
	bold := styles.Styles{Color: color, FontWeight: properties.FontWeightBold}
	start, mi := 0, 0
	for start < len(rs) {
		isMatch := mi < len(o.matched) && o.matched[mi] == start
		end := start
		for end < len(rs) && (mi < len(o.matched) && o.matched[mi] == end) == isMatch {
			if isMatch {
				mi++
			}
			end++
		}
		st := plain
		if isMatch {
			st = bold
		}
		spans = append(spans, elements.SpanText(string(rs[start:end]), st))
		start = end
	}
	return elements.RichText(styles.Styles{MaxLines: 1}, spans...)
}

// anchorBelow is the window rectangle of an element at dims just laid
// out, for a portal.
func anchorBelow(size image.Point) image.Rectangle {
	at := elements.AbsolutePosition()
	return image.Rectangle{Min: at, Max: at.Add(size)}
}
