package richedit

import (
	"image"
	"strings"

	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/richdoc"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// suggestState is the open suggestion list.
type suggestState struct {
	open    bool
	trigger rune
	query   string
	at      richdoc.Pos
	items   []Suggestion
	index   int
	clicks  []widget.Clickable
	// dismissed is the trigger Escape closed the list for.
	dismissed richdoc.Pos
}

// slashCommands are the built-in "/" commands.
func slashCommands() []Suggestion {
	kind := func(k richdoc.Kind, level int) func(*RichEditor) {
		return func(e *RichEditor) {
			if b, _ := e.Kind(); b != k {
				e.SetKind(k, level)
			}
		}
	}
	return []Suggestion{
		{Label: "Text", Detail: "Ctrl+Alt+0", Command: kind(richdoc.Paragraph, 0)},
		{Label: "Heading 1", Detail: "#", Command: kind(richdoc.Heading, 1)},
		{Label: "Heading 2", Detail: "##", Command: kind(richdoc.Heading, 2)},
		{Label: "Heading 3", Detail: "###", Command: kind(richdoc.Heading, 3)},
		{Label: "Bulleted list", Detail: "-", Command: kind(richdoc.Bullet, 0)},
		{Label: "Numbered list", Detail: "1.", Command: kind(richdoc.Numbered, 0)},
		{Label: "Checklist", Detail: "[ ]", Command: kind(richdoc.Task, 0)},
		{Label: "Quote", Detail: ">", Command: kind(richdoc.Quote, 0)},
		{Label: "Code block", Detail: "```", Command: kind(richdoc.Code, 0)},
		{Label: "Table", Command: func(e *RichEditor) { e.InsertTable(3, 3) }},
		{Label: "Divider", Detail: "---", Command: func(e *RichEditor) { e.InsertRule() }},
	}
}

// updateQuery opens, refreshes or closes the suggestion list for the
// text at the caret.
func (e *RichEditor) updateQuery() {
	s := &e.sugg
	if e.readOnly || !e.focused {
		s.open = false
		return
	}
	var triggers strings.Builder
	for r := range e.suggesters {
		triggers.WriteRune(r)
	}
	if e.slash {
		triggers.WriteRune('/')
	}
	trig, q, at, ok := e.st.Query(triggers.String(), trigSpaces(e))
	if !ok || at == s.dismissed && trig == s.trigger {
		s.open = false
		if !ok {
			s.dismissed = richdoc.Pos{Block: -1}
		}
		return
	}
	if s.open && trig == s.trigger && q == s.query && at == s.at {
		return
	}
	var items []Suggestion
	if fn := e.suggesters[trig]; fn != nil {
		items = fn(q)
	} else if trig == '/' {
		lq := strings.ToLower(q)
		for _, c := range slashCommands() {
			if strings.Contains(strings.ToLower(c.Label), lq) {
				items = append(items, c)
			}
		}
	}
	if len(items) > 8 {
		items = items[:8]
	}
	if s.trigger != trig || s.at != at {
		s.index = 0
	}
	s.trigger, s.query, s.at, s.items = trig, q, at, items
	s.index = min(s.index, max(len(items)-1, 0))
	s.open = len(items) > 0
}

// trigSpaces reports whether queries may hold spaces: mentions of full
// names do.
func trigSpaces(*RichEditor) bool { return false }

// suggestKey moves through or picks from the open list, reporting
// whether it took the key.
func (e *RichEditor) suggestKey(ev key.Event) bool {
	s := &e.sugg
	switch ev.Name {
	case key.NameUpArrow:
		s.index = (s.index + len(s.items) - 1) % len(s.items)
	case key.NameDownArrow:
		s.index = (s.index + 1) % len(s.items)
	case key.NameReturn, key.NameEnter, key.NameTab:
		return e.acceptSuggestion()
	case key.NameEscape:
		s.open = false
		s.dismissed = s.at
	default:
		return false
	}
	return true
}

// acceptSuggestion applies the highlighted suggestion.
func (e *RichEditor) acceptSuggestion() bool {
	s := &e.sugg
	if !s.open || s.index >= len(s.items) {
		return false
	}
	it := s.items[s.index]
	s.open = false
	caret := e.st.Sel.Head
	at := s.at
	switch {
	case it.Atom != nil:
		e.run(e.st.InsertAtom(*it.Atom, &at, " "))
	case it.Command != nil:
		e.run(e.st.ReplaceRange(at, caret, ""))
		it.Command(e)
	default:
		e.run(e.st.ReplaceRange(at, caret, it.Text))
	}
	e.hist.Break()
	e.wantFocus = true
	return true
}

// layoutSuggestions shows the open list under the caret.
func (e *RichEditor) layoutSuggestions(gtx layout.Context) {
	s := &e.sugg
	if !s.open {
		return
	}
	r, ok := e.caretRect(e.st.Sel.Head)
	if !ok {
		return
	}
	r = r.Add(image.Pt(0, -int(e.offY))).Add(e.origin)
	for len(s.clicks) < len(s.items) {
		s.clicks = append(s.clicks, widget.Clickable{})
	}
	items := s.items
	elements.RegisterPortalAround(gtx, r, 4, elements.PlaceAuto, func() elements.Element {
		return elements.Raw(func(gtx layout.Context) layout.Dimensions {
			return e.suggestPanel(gtx, items)
		})
	})
}

func (e *RichEditor) suggestPanel(gtx layout.Context, items []Suggestion) layout.Dimensions {
	s := &e.sugg
	rows := make([]elements.Element, 0, len(items))
	for i, it := range items {
		i, it := i, it
		click := &s.clicks[i]
		rows = append(rows, elements.Raw(func(gtx layout.Context) layout.Dimensions {
			if click.Clicked(gtx) {
				s.index = i
				e.acceptSuggestion()
				kit.RequestFrame(gtx)
			}
			return click.Layout(gtx, kit.Unselectable(func(gtx layout.Context) layout.Dimensions {
				st := styles.Styles{Padding: properties.SymmetricInset(10, 6), BorderRadius: 4, Gap: 12,
					FlexDirection: properties.FlexDirectionRow, Align: properties.AlignCenter}
				if i == s.index || click.Hovered() {
					st.Background = "theme.border"
				}
				pointer.CursorPointer.Add(gtx.Ops)
				kids := []elements.Element{elements.Text(it.Label, styles.Styles{Color: "theme.text"})}
				if it.Detail != "" {
					kids = append(kids, elements.Text(it.Detail, styles.Styles{Color: "theme.muted", FontSize: 12}))
				}
				return elements.Box(st, kids...).Layout(gtx)
			}))
		}))
	}
	gtx.Constraints.Min.X = gtx.Dp(200)
	dims := elements.Box(styles.Styles{Background: "theme.surface", BorderRadius: 8, Padding: properties.UniformInset(4),
		FlexDirection: properties.FlexDirectionColumn, BorderWidth: 1, BorderColor: "theme.border"}, rows...).Layout(gtx)
	kit.BlockDrag(gtx, dims.Size)
	return dims
}
