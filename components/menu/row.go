package menu

import (
	"image"

	"gioui.org/f32"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/mehran9675/giode/components/kit"
	"github.com/mehran9675/giode/elements"
	"github.com/mehran9675/giode/shortcut"
	"github.com/mehran9675/giode/styles"
	"github.com/mehran9675/giode/styles/properties"
)

// row returns the element for one menu row. index is the item's
// position in m.items, needed to track which row's submenu (if any)
// is currently open. width is the shared row width in pixels, as
// measured by the panel (0 while measuring).
func (m *Menu) row(item *Item, index, width int, st styles.Styles) elements.Element {
	if item.separator {
		// Divider always fills its ambient Max.X rather than hugging
		// (that's its point when used standalone), so it must be
		// wrapped to the shared row width explicitly once known -
		// otherwise it would stretch to whatever loose constraint the
		// panel happens to have been given.
		sep := m.separatorColor
		if sep == "" {
			sep = properties.CalcColorReverse(defaultSeparator)
		}
		divider := elements.Divider(styles.Styles{Color: sep, Height: 1})
		if width > 0 {
			return elements.SizedPx(width, 0, divider)
		}
		return divider
	}
	if item.disabled {
		st := styles.Styles{Padding: m.rowPadding()}
		// The color goes on the text itself: a Box's Color doesn't
		// carry over to the Text inside it, which would otherwise draw
		// in the default (dark) color.
		var label elements.Element = elements.Text(item.label, styles.Styles{Color: m.disabledText()})
		if m.hasLeading() {
			label = elements.Row(styles.Styles{Wrap: properties.WrapNoWrap}, elements.Box(styles.Styles{Width: 24}), label)
		}
		if width > 0 {
			st.Width = -1
			return elements.SizedPx(width, 0, elements.Box(st, label))
		}
		return elements.Box(st, label)
	}
	return &menuRow{
		menu:  m,
		item:  item,
		index: index,
		width: width,
		st:    st,
	}
}

// menuRow is a hoverable, clickable item row - or, when item.submenu
// is set, a hoverable row that opens a cascading nested panel instead
// of firing an action on click.
type menuRow struct {
	menu  *Menu
	item  *Item
	index int
	width int
	st    styles.Styles
}

// Layout lays out the row, handles its click, and - for a row with a
// submenu - opens/renders it while it's the active one (see
// Menu.openSubmenu).
func (r *menuRow) Layout(gtx layout.Context) layout.Dimensions {
	// Handles its input without an input pass; see kit.OpaqueInput.
	kit.OpaqueInput()
	item := r.item
	if item.submenu == nil && item.click.Clicked(gtx) {
		if item.action != nil {
			item.action()
		}
		// The whole menu: a submenu's item closes the menu it came
		// from too.
		r.menu.root().Close()
		kit.RequestFrame(gtx)
	}
	dims := item.click.Layout(gtx, kit.Unselectable(func(gtx layout.Context) layout.Dimensions {
		st := styles.Styles{
			Padding:      r.menu.rowPadding(),
			BorderRadius: 6,
		}
		// Force the row to the shared width: an unset Width makes the
		// box hug, which zeroes the constraint minimum regardless of
		// what a caller set it to. The measured width is in pixels, so
		// it's applied as exact constraints the box then expands to
		// fill (Width -1), rather than as a dp Width that would be
		// scaled a second time.
		if r.width > 0 {
			st.Width = -1
			gtx.Constraints.Min.X, gtx.Constraints.Max.X = r.width, r.width
		}
		hovered := item.click.Hovered()
		if hovered || item.click.Pressed() {
			kit.MarkLive()
		}
		if hovered && (r.menu.active != r.index || r.menu.root().keyNav) {
			// The pointer takes the highlight back from the keyboard.
			r.menu.active = r.index
			r.menu.root().keyNav = false
		}
		keyed := r.menu.root().keyNav && r.menu.active == r.index
		if hovered || keyed {
			st.Background = r.menu.hoverColor
			if st.Background == "" {
				st.Background = properties.CalcColorReverse(defaultHover)
			}
			if hovered {
				pointer.CursorPointer.Add(gtx.Ops)
				if item.submenu != nil {
					r.menu.openSubmenu = r.index
				}
			}
		}
		var children []elements.Element
		switch {
		case item.icon != nil:
			children = append(children, item.icon.Icon(styles.Styles{Color: r.st.Color, Width: 16, Height: 16}))
		case item.check == checkBox && item.checked != nil && item.checked():
			children = append(children, elements.Glyph(kit.GlyphCheck, styles.Styles{Color: r.st.Color, Width: 16}))
		case item.check == checkRadio && item.checked != nil && item.checked():
			children = append(children, radioDot(r.st.Color))
		case r.menu.hasLeading():
			children = append(children, elements.Box(styles.Styles{Width: 16, Height: 16}))
		}
		if len(children) > 0 {
			children = append(children, elements.Box(styles.Styles{Width: 8}))
		}
		// Once the shared width is known the label fills the leftover
		// space, pushing a submenu chevron to the right edge; while
		// measuring it must hug its text - and must not grow, since a
		// no-wrap row with a growing child takes all the width it's
		// offered, which would make every panel as wide as the window.
		var labelSt styles.Styles
		if r.width > 0 {
			labelSt.Width = -1
			labelSt.FlexGrow = 1
		}
		children = append(children, elements.Box(labelSt, elements.Text(item.label, styles.Styles{Color: r.st.Color})))
		if item.shortcut != "" {
			children = append(children,
				elements.Box(styles.Styles{Width: 24}),
				elements.Text(shortcut.Format(item.shortcut), styles.Styles{Color: r.menu.disabledText()}),
			)
		}
		if item.submenu != nil {
			children = append(children,
				elements.Box(styles.Styles{Width: 8}),
				elements.Glyph(kit.GlyphChevronForward, styles.Styles{Color: r.st.Color, Width: 14}),
			)
		}
		// Width -1 makes the row span the shared width, so the growing
		// label pushes a submenu chevron to the right edge.
		// No wrap: FlexGrow is ignored on wrapped lines.
		rowSt := styles.Styles{Align: properties.AlignCenter, Wrap: properties.WrapNoWrap}
		if r.width > 0 {
			rowSt.Width = -1
		}
		return elements.Box(st, elements.Row(rowSt, children...)).Layout(gtx)
	}))
	// Keep the item clickable even when a frameless window's
	// whole-window drag area happens to reach this portal - see
	// button.Layout's identical comment.
	kit.BlockDrag(gtx, dims.Size)
	node := elements.SemanticNode{Kind: elements.SemanticMenuItem, Label: item.label, Target: menuItemActor{r.menu, item}}
	if item.check != checkNone {
		node.Value = "false"
		if item.checked != nil && item.checked() {
			node.Value = "true"
		}
	}
	elements.RecordSemantic(gtx, node, dims.Size)

	// Only the real pass opens the submenu. The menu also lays its rows
	// out on scratch passes with input disabled (measureWidth, and
	// RegisterPortal's sizing pass); registering the submenu's portal
	// there anchors it at the scratch origin, drawing a second copy in
	// the window's top-left corner.
	if item.submenu != nil && r.menu.openSubmenu == r.index && gtx.Source.Enabled() {
		// Shift up by the submenu's top padding so its first row lines
		// up with this one.
		pos := elements.AbsolutePosition()
		anchor := image.Pt(r.menu.submenuX(gtx, item.submenu, pos.X+dims.Size.X), pos.Y-gtx.Dp(unit.Dp(item.submenu.panelPaddingTop())))
		item.submenu.OpenAt(f32.Pt(float32(anchor.X), float32(anchor.Y)))
		item.submenu.Layout(gtx)
	}
	return dims
}

// submenuX places a submenu beside this menu, as Windows and macOS do:
// to the right of the row (rowRight, in window pixels) when it fits,
// otherwise with its right edge at this panel's left edge, rather than
// moved over this menu's rows. A cascade keeps the direction it took:
// a submenu that opened to the left opens its own to the left first.
// When neither side fits it takes the preferred side and the window
// keeps it on screen.
func (m *Menu) submenuX(gtx layout.Context, sub *Menu, rowRight int) int {
	win := elements.WindowSize()
	if win == (image.Point{}) {
		win = gtx.Constraints.Max
	}
	mgtx := gtx
	mgtx.Constraints = layout.Constraints{Max: win}
	w := elements.Measure(mgtx, elements.Raw(sub.panel)).Size.X
	margin := gtx.Dp(unit.Dp(kit.Margin))
	right, left := rowRight, m.panelLeft-w
	fitsRight, fitsLeft := right+w <= win.X-margin, left >= margin
	// A root menu's submenus open on its reading side: right, or left
	// right to left; deeper ones keep the direction their parent took.
	toLeft := m.opensLeft || (m.parent == nil && kit.RTL(gtx))
	switch {
	case toLeft && !fitsLeft && fitsRight:
		toLeft = false
	case !toLeft && !fitsRight && fitsLeft:
		toLeft = true
	}
	sub.opensLeft = toLeft
	if toLeft {
		return left
	}
	return right
}

// radioDot is a selected radio item's indicator: a dot in the 16dp
// leading column.
func radioDot(c properties.Color) elements.Element {
	return elements.RawFunc(func(gtx layout.Context) layout.Dimensions {
		sz := gtx.Dp(16)
		d := gtx.Dp(6)
		o := (sz - d) / 2
		defer clip.Ellipse(image.Rect(o, o, o+d, o+d)).Push(gtx.Ops).Pop()
		paint.Fill(gtx.Ops, elements.ResolveColor(c))
		return layout.Dimensions{Size: image.Pt(sz, sz)}
	})
}

// menuItemActor lets automation activate a menu item as a click
// would.
type menuItemActor struct {
	menu *Menu
	item *Item
}

func (a menuItemActor) Click() {
	if a.item.submenu != nil {
		return
	}
	if a.item.action != nil {
		a.item.action()
	}
	a.menu.root().Close()
}

// disabledText is the text color of disabled rows.
func (m *Menu) disabledText() properties.Color {
	if m.disabledColor != "" {
		return m.disabledColor
	}
	return properties.CalcColorReverse(defaultDisabled)
}
