// Package palette provides a command palette: a search box over the
// app's commands, opened with Mod+K (Command+K on Apple systems,
// Ctrl+K elsewhere), that runs the one picked. Matching is fuzzy
// ("nwtb" finds "New tab"), recent commands come first, and each
// command's own shortcut shows beside it.
//
//	p := palette.New(
//		palette.Command{ID: "new-tab", Title: "New tab", Shortcut: "Mod+T", Run: newTab},
//		palette.Command{ID: "theme", Title: "Toggle dark mode", Group: "View", Run: toggleTheme},
//	)
//	// In the view, every frame:
//	giode.Shortcuts(p.Shortcuts()...)
//	giode.Stack(..., p)
package palette

import (
	"image"

	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/xmhrnx/giode/components/combobox"
	"github.com/xmhrnx/giode/components/input"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/shortcut"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Command is something the palette runs.
type Command struct {
	// ID identifies it, for recents; Title is shown and searched.
	ID, Title string
	// Group puts it under a heading.
	Group string
	// Shortcut is its keyboard shortcut ("Mod+T"), shown beside it and
	// registered by Shortcuts.
	Shortcut string
	// Keywords are also searched: synonyms ("quit" for Close window).
	Keywords []string
	Icon     elements.Element
	Disabled bool
	Run      func()
}

// Palette is a command palette. Create it once with New, put it in the
// view (it takes no space while closed) and register its Shortcuts.
type Palette struct {
	commands []Command
	recents  []string
	open     bool
	field    *input.Input
	list     combobox.List
	last     string
	key      string
	scrim    int
	// MaxRecents is how many recent commands lead the list: 5 by
	// default.
	MaxRecents int
}

// New returns a palette of cmds.
func New(cmds ...Command) *Palette {
	p := &Palette{commands: cmds, key: "Mod+K", MaxRecents: 5}
	p.field = input.New("Type a command…", styles.Styles{BorderColor: "#00000000", Padding: properties.SymmetricInset(14, 12), FontSize: 16})
	p.field.OnKeys(p.onKey, key.NameUpArrow, key.NameDownArrow, key.NameReturn, key.NameEnter, key.NameEscape, key.NamePageUp, key.NamePageDown)
	p.list.OnPick = func(o combobox.Option) { p.run(o.Value) }
	p.list.EmptyText = "No matching commands"
	p.list.MaxRows = 10
	p.list.Plain = true
	return p
}

// SetCommands replaces the commands.
func (p *Palette) SetCommands(cmds ...Command) { p.commands = cmds; p.refresh() }

// Add adds commands.
func (p *Palette) Add(cmds ...Command) { p.commands = append(p.commands, cmds...); p.refresh() }

// Key changes the shortcut that opens it from Mod+K.
func (p *Palette) Key(spec string) *Palette { p.key = spec; return p }

// Placeholder sets the search box's placeholder.
func (p *Palette) Placeholder(s string) *Palette { p.field.Placeholder(s); return p }

// Open shows it, with an empty search.
func (p *Palette) Open() {
	p.open = true
	p.field.SetText("")
	p.last = ""
	p.refresh()
	p.field.Focus()
}

// Close hides it.
func (p *Palette) Close() { p.open = false; p.field.Blur() }

// Toggle opens or closes it.
func (p *Palette) Toggle() {
	if p.open {
		p.Close()
	} else {
		p.Open()
	}
}

// Opened reports whether it shows.
func (p *Palette) Opened() bool { return p.open }

// Shortcuts are the palette's key and every command's shortcut, for
// giode.Shortcuts.
func (p *Palette) Shortcuts() []shortcut.Shortcut {
	out := []shortcut.Shortcut{shortcut.New(p.key, p.Toggle)}
	for _, c := range p.commands {
		if c.Shortcut == "" || c.Disabled || c.Run == nil {
			continue
		}
		if _, _, err := shortcut.Parse(c.Shortcut); err != nil {
			continue
		}
		id := c.ID
		out = append(out, shortcut.New(c.Shortcut, func() { p.run(id) }))
	}
	return out
}

// Run runs the command id as if picked, and counts it as recent.
func (p *Palette) Run(id string) { p.run(id) }

func (p *Palette) find(id string) (Command, bool) {
	for _, c := range p.commands {
		if c.ID == id {
			return c, true
		}
	}
	return Command{}, false
}

func (p *Palette) run(id string) {
	c, ok := p.find(id)
	if !ok || c.Disabled {
		return
	}
	p.Close()
	rec := []string{id}
	for _, r := range p.recents {
		if r != id {
			rec = append(rec, r)
		}
	}
	if max := p.MaxRecents; len(rec) > max {
		rec = rec[:max]
	}
	p.recents = rec
	if c.Run != nil {
		c.Run()
	}
}

// Recents are the IDs of the commands last run, latest first, to save
// and restore with SetRecents.
func (p *Palette) Recents() []string { return append([]string(nil), p.recents...) }

// SetRecents restores the recent commands.
func (p *Palette) SetRecents(ids []string) { p.recents = append([]string(nil), ids...) }

func option(c Command, group string) combobox.Option {
	o := combobox.Option{Value: c.ID, Label: c.Title, Group: group, Keywords: c.Keywords, Icon: c.Icon, Disabled: c.Disabled}
	if c.Shortcut != "" {
		o.Detail = shortcut.Format(c.Shortcut)
	}
	return o
}

// refresh lists the commands for the query: recents first when it's
// empty.
func (p *Palette) refresh() {
	q := p.field.Text()
	var opts []combobox.Option
	if q == "" {
		seen := map[string]bool{}
		for _, id := range p.recents {
			if c, ok := p.find(id); ok {
				opts = append(opts, option(c, "Recent"))
				seen[id] = true
			}
		}
		for _, c := range p.commands {
			if !seen[c.ID] {
				opts = append(opts, option(c, c.Group))
			}
		}
		p.list.SetItems(opts)
		return
	}
	for _, c := range p.commands {
		opts = append(opts, option(c, ""))
	}
	p.list.SetItems(combobox.Filter(opts, q))
}

func (p *Palette) onKey(e key.Event) {
	switch e.Name {
	case key.NameUpArrow:
		p.list.Move(-1)
	case key.NameDownArrow:
		p.list.Move(1)
	case key.NamePageUp:
		p.list.First()
	case key.NamePageDown:
		p.list.Last()
	case key.NameReturn, key.NameEnter:
		if o, ok := p.list.Selected(); ok {
			p.run(o.Value)
		}
	case key.NameEscape:
		p.Close()
	}
}

// Layout shows the palette while open, over the window; closed, it
// takes no space.
func (p *Palette) Layout(gtx layout.Context) layout.Dimensions {
	if !p.open {
		return layout.Dimensions{}
	}
	if t := p.field.Text(); t != p.last {
		p.last = t
		p.refresh()
	}
	elements.HandleDismiss(p.Close)
	elements.RegisterOverlay(func() elements.Element { return elements.Raw(p.layoutOverlay) })
	return layout.Dimensions{}
}

// OutOfFlow marks it as taking no space in its container.
func (p *Palette) OutOfFlow() {}

func (p *Palette) layoutOverlay(gtx layout.Context) layout.Dimensions {
	win := gtx.Constraints.Max
	// The scrim: dims the window and closes the palette on a press.
	for {
		ev, ok := gtx.Source.Event(pointer.Filter{Target: &p.scrim, Kinds: pointer.Press})
		if !ok {
			break
		}
		if _, ok := ev.(pointer.Event); ok {
			p.Close()
			kit.RequestFrame(gtx)
		}
	}
	area := clip.Rect{Max: win}.Push(gtx.Ops)
	paint.ColorOp{Color: properties.CalcColor("#00000066")}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	event.Op(gtx.Ops, &p.scrim)
	area.Pop()

	width := min(gtx.Dp(560), win.X-gtx.Dp(32))
	x := (win.X - width) / 2
	y := min(gtx.Dp(80), win.Y/8)
	defer op.Offset(image.Pt(x, y)).Push(gtx.Ops).Pop()
	gtx.Constraints = layout.Constraints{Min: image.Pt(width, 0), Max: image.Pt(width, win.Y-y-gtx.Dp(16))}
	panel := elements.Box(styles.Styles{
		FlexDirection: properties.FlexDirectionColumn, Width: -1, Background: "theme.surface", BorderRadius: 12,
		BorderWidth: 1, BorderColor: "theme.border", BoxShadow: properties.BoxShadow{Y: 12, Blur: 32, Color: "#00000055"},
	},
		p.field,
		elements.Box(styles.Styles{Height: 1, Width: -1, Background: "theme.border"}),
		elements.Raw(func(gtx layout.Context) layout.Dimensions { return p.list.Layout(gtx, gtx.Constraints.Max.X) }),
	)
	dims := panel.Layout(gtx)
	// Presses on the panel don't reach the scrim.
	defer clip.Rect{Max: dims.Size}.Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, p)
	kit.BlockDrag(gtx, dims.Size)
	return layout.Dimensions{Size: win}
}
