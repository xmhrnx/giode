// Package emojipicker provides an emoji picker: a searchable grid of
// Unicode's emoji by category, with skin tones and recently used ones
// first. Show it in a popover or a sheet next to a message field.
//
//	p := emojipicker.New().OnPick(func(e string) { field.InsertText(e) })
package emojipicker

import (
	"image"
	"strings"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/xmhrnx/giode/components/combobox"
	"github.com/xmhrnx/giode/components/input"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

type emojiData struct {
	group uint8
	e     string
	name  string
	tones string
}

// Emoji is one emoji of the picker's set.
type Emoji struct {
	Emoji, Name, Group string
	// SkinTones are its five skin tone variants, light to dark; nil
	// when it has none.
	SkinTones []string
}

// All returns every emoji the picker offers, in Unicode's order.
func All() []Emoji {
	out := make([]Emoji, len(emojis))
	for i, d := range emojis {
		out[i] = d.public()
	}
	return out
}

func (d emojiData) public() Emoji {
	e := Emoji{Emoji: d.e, Name: d.name, Group: groupNames[d.group]}
	if d.tones != "" {
		e.SkinTones = strings.Split(d.tones, "|")
	}
	return e
}

// withTone is e in skin tone t (1 light to 5 dark), or as is.
func (d emojiData) withTone(t int) string {
	if t < 1 || t > 5 || d.tones == "" {
		return d.e
	}
	return strings.Split(d.tones, "|")[t-1]
}

// Picker is the emoji picker. Create it once with New and lay it out
// every frame; it fills the width it's given, 8 rows of emoji tall.
type Picker struct {
	search  *input.Input
	onPick  func(string)
	tone    int
	recents []int // indexes into emojis, latest first
	// MaxRecents is how many recent emoji show first: 24 by default.
	MaxRecents int

	clicks   map[int]*widget.Clickable
	tabs     []widget.Clickable
	toneBtns [6]widget.Clickable
	list     layout.List
	lastQ    string
	results  []int
	jump     int // a group to scroll to, +1
	rowsOf   []int
}

// New returns a picker.
func New() *Picker {
	p := &Picker{clicks: map[int]*widget.Clickable{}, MaxRecents: 24}
	p.search = input.New("Search emoji")
	p.search.Submit(func(string) {
		if len(p.results) > 0 {
			p.pick(p.results[0])
		}
	})
	p.tabs = make([]widget.Clickable, len(groupNames)+1)
	return p
}

// OnPick sets the handler called with each emoji picked, in the chosen
// skin tone.
func (p *Picker) OnPick(fn func(emoji string)) *Picker {
	p.onPick = fn
	return p
}

// SkinTone sets the skin tone emoji are offered in: 0 for none (the
// default yellow), 1 light to 5 dark.
func (p *Picker) SkinTone(t int) *Picker {
	p.tone = max(0, min(t, 5))
	return p
}

// Tone is the chosen skin tone, to save.
func (p *Picker) Tone() int { return p.tone }

// Recents are the emoji last picked, latest first, to save and restore
// with SetRecents.
func (p *Picker) Recents() []string {
	out := make([]string, len(p.recents))
	for i, r := range p.recents {
		out[i] = emojis[r].e
	}
	return out
}

// SetRecents restores the recent emoji.
func (p *Picker) SetRecents(list []string) {
	p.recents = nil
	for _, e := range list {
		for i, d := range emojis {
			if d.e == e {
				p.recents = append(p.recents, i)
				break
			}
		}
	}
}

// Search returns the emoji whose names match query, best first.
func Search(query string) []Emoji {
	var out []Emoji
	for _, i := range search(query) {
		out = append(out, emojis[i].public())
	}
	return out
}

func search(q string) []int {
	q = strings.TrimSpace(strings.ToLower(q))
	if q == "" {
		return nil
	}
	type hit struct{ i, score int }
	var hits []hit
	for i, d := range emojis {
		// Every word must start a word of the name; fuzzy otherwise.
		if strings.Contains(" "+d.name, " "+q) {
			hits = append(hits, hit{i, 1000 - len(d.name)})
		} else if sc, _, ok := combobox.Match(q, d.name); ok && sc > 10*len([]rune(q)) {
			hits = append(hits, hit{i, sc})
		}
	}
	// Stable sort by score, keeping Unicode's order between equals.
	for i := 1; i < len(hits); i++ {
		for j := i; j > 0 && hits[j].score > hits[j-1].score; j-- {
			hits[j], hits[j-1] = hits[j-1], hits[j]
		}
	}
	out := make([]int, len(hits))
	for i, h := range hits {
		out[i] = h.i
	}
	return out
}

func (p *Picker) pick(i int) {
	rec := []int{i}
	for _, r := range p.recents {
		if r != i {
			rec = append(rec, r)
		}
	}
	if len(rec) > p.MaxRecents {
		rec = rec[:p.MaxRecents]
	}
	p.recents = rec
	if p.onPick != nil {
		p.onPick(emojis[i].withTone(p.tone))
	}
}

// section is a heading and its emoji.
type section struct {
	title string
	items []int
	group int // -1 for recents and results
}

func (p *Picker) sections() []section {
	if q := p.search.Text(); q != "" {
		if q != p.lastQ {
			p.lastQ = q
			p.results = search(q)
		}
		return []section{{title: "Results", items: p.results, group: -1}}
	}
	p.lastQ, p.results = "", nil
	var out []section
	if len(p.recents) > 0 {
		out = append(out, section{title: "Recently used", items: p.recents, group: -1})
	}
	start := 0
	for g := range groupNames {
		end := start
		for end < len(emojis) && int(emojis[end].group) == g {
			end++
		}
		items := make([]int, end-start)
		for k := range items {
			items[k] = start + k
		}
		out = append(out, section{title: groupNames[g], items: items, group: g})
		start = end
	}
	return out
}

// Layout lays out the picker.
func (p *Picker) Layout(gtx layout.Context) layout.Dimensions {
	cell := gtx.Dp(36)
	width := gtx.Constraints.Max.X
	cols := max(width/cell, 4)
	secs := p.sections()

	// Rows: a heading or a row of emoji.
	type row struct {
		heading string
		items   []int
	}
	var rows []row
	p.rowsOf = p.rowsOf[:0]
	for _, s := range secs {
		p.rowsOf = append(p.rowsOf, len(rows))
		rows = append(rows, row{heading: s.title})
		for k := 0; k < len(s.items); k += cols {
			rows = append(rows, row{items: s.items[k:min(k+cols, len(s.items))]})
		}
	}
	if p.jump > 0 {
		// A tab: scroll to its group's heading.
		g := p.jump - 1
		p.jump = 0
		for si, s := range secs {
			if s.group == g {
				p.list.Position = layout.Position{First: p.rowsOf[si]}
			}
		}
	}

	grid := elements.Raw(func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.Y = min(gtx.Constraints.Max.Y, cell*8)
		gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
		p.list.Axis = layout.Vertical
		if len(rows) == 1 {
			return elements.Box(styles.Styles{Padding: properties.UniformInset(16)},
				elements.Text("No emoji found", styles.Styles{Color: "theme.muted"})).Layout(gtx)
		}
		return p.list.Layout(gtx, len(rows), func(gtx layout.Context, ri int) layout.Dimensions {
			r := rows[ri]
			if r.items == nil {
				return elements.Box(styles.Styles{Padding: properties.Inset{Top: 8, Bottom: 4, Left: 6}},
					elements.Text(r.heading, styles.Styles{Color: "theme.muted", FontSize: 12, FontWeight: properties.FontWeightSemiBold})).Layout(gtx)
			}
			kids := make([]elements.Element, len(r.items))
			for k, i := range r.items {
				kids[k] = p.cell(i, cell)
			}
			return elements.Row(styles.Styles{}, kids...).Layout(gtx)
		})
	})

	tabs := make([]elements.Element, 0, len(groupNames))
	for g := range groupNames {
		g := g
		click := &p.tabs[g]
		first := emojis[p.firstOf(g)].e
		tabs = append(tabs, elements.Raw(func(gtx layout.Context) layout.Dimensions {
			if click.Clicked(gtx) {
				p.search.SetText("")
				p.jump = g + 1
				kit.RequestFrame(gtx)
			}
			return click.Layout(gtx, kit.Unselectable(func(gtx layout.Context) layout.Dimensions {
				pointer.CursorPointer.Add(gtx.Ops)
				st := styles.Styles{Padding: properties.SymmetricInset(6, 4), BorderRadius: 6}
				if click.Hovered() {
					st.Background = "theme.border"
				}
				return elements.Box(st, elements.Accessible(elements.Text(first, styles.Styles{FontSize: 16}),
					func(n *elements.SemanticNode) { n.Kind, n.Label = elements.SemanticButton, groupNames[g] })).Layout(gtx)
			}))
		}))
	}

	tones := make([]elements.Element, 6)
	toneNames := []string{"Default skin tone", "Light", "Medium-light", "Medium", "Medium-dark", "Dark"}
	for t := 0; t < 6; t++ {
		t := t
		click := &p.toneBtns[t]
		hand := emojiData{e: "✋", tones: "✋🏻|✋🏼|✋🏽|✋🏾|✋🏿"}.withTone(t)
		tones[t] = elements.Raw(func(gtx layout.Context) layout.Dimensions {
			if click.Clicked(gtx) {
				p.tone = t
				kit.RequestFrame(gtx)
			}
			return click.Layout(gtx, kit.Unselectable(func(gtx layout.Context) layout.Dimensions {
				pointer.CursorPointer.Add(gtx.Ops)
				st := styles.Styles{Padding: properties.UniformInset(3), BorderRadius: 6, BorderWidth: 1, BorderColor: "#00000000"}
				if p.tone == t {
					st.BorderColor = "theme.primary"
				}
				return elements.Box(st, elements.Accessible(elements.Text(hand, styles.Styles{FontSize: 14}),
					func(n *elements.SemanticNode) { n.Kind, n.Label = elements.SemanticRadio, toneNames[t] })).Layout(gtx)
			}))
		})
	}

	return elements.Box(styles.Styles{FlexDirection: properties.FlexDirectionColumn, Width: -1, Gap: 6,
		Padding: properties.UniformInset(8), Background: "theme.surface", BorderRadius: 10, BorderWidth: 1, BorderColor: "theme.border"},
		p.search,
		elements.Row(styles.Styles{Gap: 2, Wrap: properties.WrapWrap}, tabs...),
		grid,
		elements.Row(styles.Styles{Gap: 4, Align: properties.AlignCenter},
			append([]elements.Element{elements.Text("Skin tone", styles.Styles{Color: "theme.muted", FontSize: 12})}, tones...)...),
	).Layout(gtx)
}

func (p *Picker) firstOf(g int) int {
	for i, d := range emojis {
		if int(d.group) == g {
			return i
		}
	}
	return 0
}

func (p *Picker) cell(i, size int) elements.Element {
	click := p.clicks[i]
	if click == nil {
		click = new(widget.Clickable)
		p.clicks[i] = click
	}
	e := emojis[i].withTone(p.tone)
	name := emojis[i].name
	return elements.Raw(func(gtx layout.Context) layout.Dimensions {
		if click.Clicked(gtx) {
			p.pick(i)
			kit.RequestFrame(gtx)
		}
		return click.Layout(gtx, kit.Unselectable(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints = layout.Exact(image.Pt(size, size))
			pointer.CursorPointer.Add(gtx.Ops)
			st := styles.Styles{Width: -1, Height: -1, Align: properties.AlignCenter, Justify: properties.JustifyCenter, BorderRadius: 6}
			if click.Hovered() {
				st.Background = "theme.border"
			}
			return elements.Box(st, elements.Accessible(elements.Text(e, styles.Styles{FontSize: 22}),
				func(n *elements.SemanticNode) { n.Kind, n.Label = elements.SemanticButton, name })).Layout(gtx)
		}))
	})
}
