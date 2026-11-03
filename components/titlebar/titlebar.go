// Package titlebar provides a custom window title bar - drag handle,
// title, optional icon and menus, and minimize/maximize/close buttons
// - for frameless windows that replace the operating system's own.
package titlebar

import (
	"image"
	"image/color"
	"slices"

	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/i18n"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Window is what a title bar controls. *giode.App implements it.
type Window interface {
	Minimize()
	ToggleMaximize()
	Close()
	Maximized() bool
	Focused() bool
}

// Variant selects the look and button placement.
type Variant string

const (
	// VariantWindows places square minimize, maximize and close
	// buttons on the right; close turns red on hover. The default.
	VariantWindows Variant = "windows"
	// VariantMac places round close, minimize and maximize "traffic
	// light" buttons on the left and centers the title.
	VariantMac Variant = "mac"
)

// Button identifies a window button.
type Button int

const (
	ButtonMinimize Button = iota
	ButtonMaximize
	ButtonClose
)

var (
	defaultBackground = "theme.surface"
	defaultText       = "theme.text"
	defaultInactive   = "theme.muted"
	closeHover        = "#e81123"
	closePressed      = "#f1707a"
	// Traffic light colors, indexed by Button.
	macColors = [3]string{ButtonMinimize: "#febc2e", ButtonMaximize: "#28c840", ButtonClose: "#ff5f57"}
	// Traffic light glyphs, shown while the group is hovered.
	macGlyphs = [3]kit.Glyph{ButtonMinimize: kit.GlyphMinus, ButtonMaximize: kit.GlyphPlus, ButtonClose: kit.GlyphClose}
)

// TitleBar is a stateful custom title bar. Create it once with New and
// lay it out every frame at the top of the window:
//
//	bar := giode.TitleBar(app, "My app")
//	app.Run(func() giode.Element {
//		return giode.Stack(giode.Styles{Width: -1, Height: -1}, bar, content())
//	})
//
// Use it with Config{Frameless: true} (and without Draggable, so only
// the bar drags the window). The bar's empty space drags the window;
// on Windows it is also a real caption, so double-clicking maximizes
// and dragging to a screen edge snaps. Everything placed in it -
// buttons, menus - stays clickable.
type TitleBar struct {
	win     Window
	title   string
	icon    elements.Element
	st      styles.Styles
	variant Variant
	height  unit.Dp

	leading, trailing []elements.Element

	hidden  [3]bool
	buttons [3]widget.Clickable
	onClose func() bool

	dbl gesture.Click
}

// New returns a title bar for win. The styles argument is optional:
// Background (default theme.surface), Color (title and glyphs, default
// theme.text), Height (default 36dp, 28dp for VariantMac), FontSize,
// FontWeight, Padding, BorderColor/BorderWidth (a line under the bar).
func New(win Window, title string, st ...styles.Styles) *TitleBar {
	t := &TitleBar{win: win, title: title, variant: VariantWindows}
	if len(st) > 0 {
		t.st = st[0]
	}
	return t
}

// Styles replaces the bar's styles (see New) - solid in a window, a
// transparent fade over fullscreen video. It returns t for chaining.
func (t *TitleBar) Styles(st styles.Styles) *TitleBar {
	t.st = st
	return t
}

// Title replaces the title. It returns t for chaining.
func (t *TitleBar) Title(title string) *TitleBar {
	t.title = title
	return t
}

// Icon sets an element (typically an icon) shown before the title. It
// returns t for chaining.
func (t *TitleBar) Icon(icon elements.Element) *TitleBar {
	t.icon = icon
	return t
}

// Leading adds elements after the title - menus (File, Edit, ...),
// tabs, a search field. They stay clickable inside the drag area. It
// returns t for chaining.
func (t *TitleBar) Leading(els ...elements.Element) *TitleBar {
	t.leading = append(t.leading, els...)
	return t
}

// Trailing adds elements just before the window buttons - an account
// button, a theme switch. It returns t for chaining.
func (t *TitleBar) Trailing(els ...elements.Element) *TitleBar {
	t.trailing = append(t.trailing, els...)
	return t
}

// Variant sets the look and button placement. It returns t for
// chaining.
func (t *TitleBar) Variant(v Variant) *TitleBar {
	t.variant = v
	return t
}

// Hide removes window buttons, e.g. Hide(ButtonMaximize) for a window
// that shouldn't be maximized. It returns t for chaining.
func (t *TitleBar) Hide(buttons ...Button) *TitleBar {
	for _, b := range buttons {
		if b >= 0 && int(b) < len(t.hidden) {
			t.hidden[b] = true
		}
	}
	return t
}

// OnClose sets a handler run when the close button is clicked; return
// false to keep the window open (to ask about unsaved changes, say).
// Otherwise the button then closes the window as the system's would:
// to the tray with Config.CloseToTray, else ending the app.
// It returns t for chaining.
func (t *TitleBar) OnClose(fn func() bool) *TitleBar {
	t.onClose = fn
	return t
}

// Press performs a window button's action, as a click would.
func (t *TitleBar) Press(b Button) {
	switch b {
	case ButtonMinimize:
		t.win.Minimize()
	case ButtonMaximize:
		t.win.ToggleMaximize()
	case ButtonClose:
		if t.onClose != nil && !t.onClose() {
			return
		}
		// Close as the system's close button would: a window that closes
		// to the tray hides there (*giode.App's RequestClose).
		if r, ok := t.win.(interface{ RequestClose() }); ok {
			r.RequestClose()
		} else {
			t.win.Close()
		}
	}
}

func (t *TitleBar) barHeight() unit.Dp {
	if t.st.Height > 0 {
		return unit.Dp(t.st.Height)
	}
	if t.variant == VariantMac {
		return 28
	}
	return 36
}

// Layout lays out the bar across the available width.
func (t *TitleBar) Layout(gtx layout.Context) layout.Dimensions {
	// Handles its input without an input pass; see kit.OpaqueInput.
	kit.OpaqueInput()
	for i := range t.buttons {
		if t.buttons[i].Clicked(gtx) {
			t.Press(Button(i))
			kit.RequestFrame(gtx)
		}
	}
	// Double-clicking the bar maximizes. Windows handles this itself
	// (the bar is a real caption there); this covers the platforms
	// that only implement dragging.
	for {
		ev, ok := t.dbl.Update(gtx.Source)
		if !ok {
			break
		}
		if ev.Kind == gesture.KindClick && ev.NumClicks == 2 {
			t.win.ToggleMaximize()
		}
	}

	st := styles.Merge(styles.Styles{
		Background: defaultBackground,
		Color:      defaultText,
		FontSize:   13,
	}, t.st)
	focused := t.win.Focused()
	textColor := st.Color
	if !focused {
		textColor = defaultInactive
		if t.st.Color != "" {
			textColor = kit.Alpha(t.st.Color, 0.6)
		}
	}

	h := gtx.Dp(t.barHeight())
	size := image.Pt(gtx.Constraints.Max.X, h)
	if size.X >= 1e6 {
		size.X = gtx.Constraints.Min.X
	}
	gtx.Constraints = layout.Exact(size)

	// Background and bottom border.
	properties.PaintBackground(gtx.Ops, size, st.Background, 0)
	if bw := gtx.Dp(unit.Dp(st.BorderWidth)); bw > 0 && st.BorderColor != "" {
		paint.FillShape(gtx.Ops, properties.CalcColor(st.BorderColor), clip.Rect{Min: image.Pt(0, h-bw), Max: size}.Op())
	}

	// The double-click listener sits under the drag handle, which must
	// be the topmost area for the platform to see it; the handle comes
	// before the contents, which carve themselves out of it.
	cl := clip.Rect{Max: size}.Push(gtx.Ops)
	t.dbl.Add(gtx.Ops)
	cl.Pop()
	kit.DragArea(gtx, size)

	pad := st.Padding
	if pad == (properties.Inset{}) {
		pad = properties.SymmetricInset(12, 0)
	}
	// The window buttons sit flush with the right edge, outside the
	// padding, as on Windows; the mac ones are part of the row.
	bw := 0
	if t.variant != VariantMac {
		bw = t.windowsButtonsWidth(gtx)
	}
	rowW := max(0, size.X-bw)

	// Everything but the title keeps its natural width; the title gets
	// what's left and ends in an ellipsis, so a long one never pushes
	// the menus under the window buttons.
	gap := gtx.Dp(8)
	mgtx := gtx
	mgtx.Constraints = layout.Constraints{Max: image.Pt(rowW, h)}
	measure := func(els []elements.Element) (w, n int) {
		for _, el := range els {
			if el == nil {
				continue
			}
			w += elements.Measure(mgtx, el).Size.X
			n++
		}
		return w, n
	}
	var left, right []elements.Element
	if t.variant == VariantMac {
		left = append(left, t.macButtons(focused))
	}
	if t.icon != nil {
		left = append(left, t.icon)
	}
	left = append(left, t.leading...)
	right = append(right, t.trailing...)
	leftW, leftN := measure(left)
	rightW, rightN := measure(right)
	padX := gtx.Dp(unit.Dp(pad.Left)) + gtx.Dp(unit.Dp(pad.Right))

	var titleMax int
	if t.variant == VariantMac {
		// Centered over the bar, clear of both sides.
		side := max(gtx.Dp(unit.Dp(pad.Left))+leftW+leftN*gap, gtx.Dp(unit.Dp(pad.Right))+rightW+rightN*gap)
		titleMax = max(0, size.X-2*side)
	} else {
		// The title, then the leading elements, a spacer and the
		// trailing ones: one gap after each element but the last.
		n := leftN + rightN + 2
		titleMax = max(0, rowW-padX-leftW-rightW-(n-1)*gap-gtx.Dp(8))
	}
	titleText := elements.Text(t.title, styles.Styles{
		Color: textColor, FontSize: st.FontSize, FontWeight: st.FontWeight,
		TextOverflow: properties.TextOverflowEllipsis, MaxLines: 1,
	})
	titleBox := elements.Raw(func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = 0
		gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, titleMax)
		return titleText.Layout(gtx)
	})

	var row []elements.Element
	if t.variant == VariantMac {
		row = append(row, t.macButtons(focused))
	}
	if t.icon != nil {
		row = append(row, noDrag(t.icon))
	}
	if t.variant == VariantMac {
		// The title is centered over the whole bar below; reserve the
		// flexible space here.
		row = append(row, t.wrap(t.leading)...)
		row = append(row, elements.Box(styles.Styles{FlexGrow: 1}))
	} else {
		row = append(row, elements.Box(styles.Styles{Padding: properties.Inset{End: 8}}, titleBox))
		row = append(row, t.wrap(t.leading)...)
		row = append(row, elements.Box(styles.Styles{FlexGrow: 1}))
	}
	row = append(row, t.wrap(t.trailing)...)

	content := elements.Row(styles.Styles{
		Width: -1, Height: -1, Align: properties.AlignCenter, Gap: 8,
		Wrap: properties.WrapNoWrap, Padding: pad,
	}, row...)

	if t.variant == VariantMac {
		content.Layout(gtx)
		// Centered title, painted over the empty middle.
		macro := op.Record(gtx.Ops)
		tgtx := gtx
		tgtx.Constraints = layout.Constraints{Max: image.Pt(titleMax, h)}
		td := titleBox.Layout(tgtx)
		call := macro.Stop()
		off := op.Offset(image.Pt((size.X-td.Size.X)/2, (h-td.Size.Y)/2)).Push(gtx.Ops)
		call.Add(gtx.Ops)
		off.Pop()
	} else {
		// Right to left, the buttons sit at the left edge and the row
		// (mirrored by its Box) fills the rest.
		rowX, btnX := 0, rowW
		if kit.RTL(gtx) {
			rowX, btnX = bw, 0
		}
		func() {
			off := op.Offset(image.Pt(rowX, 0)).Push(gtx.Ops)
			defer off.Pop()
			defer elements.PushOffset(image.Pt(rowX, 0))()
			cgtx := gtx
			cgtx.Constraints = layout.Exact(image.Pt(rowW, h))
			content.Layout(cgtx)
		}()
		off := op.Offset(image.Pt(btnX, 0)).Push(gtx.Ops)
		pop := elements.PushOffset(image.Pt(btnX, 0))
		t.windowsButtons(gtx, h, textColor)
		pop()
		off.Pop()
	}
	elements.RecordSemantic(gtx, elements.SemanticNode{Kind: elements.SemanticText, Label: t.title}, size)
	return layout.Dimensions{Size: size}
}

// wrap makes custom elements non-draggable over their whole area.
func (t *TitleBar) wrap(els []elements.Element) []elements.Element {
	out := make([]elements.Element, len(els))
	for i, el := range els {
		out[i] = noDrag(el)
	}
	return out
}

func noDrag(el elements.Element) elements.Element {
	return elements.Raw(func(gtx layout.Context) layout.Dimensions {
		dims := el.Layout(gtx)
		kit.BlockDrag(gtx, dims.Size)
		return dims
	})
}

// --- Windows-style buttons ---------------------------------------------------

const winButtonWidth = 46

func (t *TitleBar) visible() []Button {
	var out []Button
	for _, b := range []Button{ButtonMinimize, ButtonMaximize, ButtonClose} {
		if !t.hidden[b] {
			out = append(out, b)
		}
	}
	return out
}

func (t *TitleBar) windowsButtonsWidth(gtx layout.Context) int {
	return len(t.visible()) * gtx.Dp(winButtonWidth)
}

func (t *TitleBar) windowsButtons(gtx layout.Context, h int, glyphColor properties.Color) {
	w := gtx.Dp(winButtonWidth)
	x := 0
	buttons := t.visible()
	if kit.RTL(gtx) {
		// Mirrored, as Windows does for right-to-left apps: close
		// first, at the left edge.
		slices.Reverse(buttons)
	}
	for _, b := range buttons {
		off := op.Offset(image.Pt(x, 0)).Push(gtx.Ops)
		pop := elements.PushOffset(image.Pt(x, 0))
		click := &t.buttons[b]
		bgtx := gtx
		bgtx.Constraints = layout.Exact(image.Pt(w, h))
		click.Layout(bgtx, func(gtx layout.Context) layout.Dimensions {
			sz := image.Pt(w, h)
			fg := properties.CalcColor(glyphColor)
			var bg color.NRGBA
			switch {
			case b == ButtonClose && click.Pressed():
				bg, fg = properties.CalcColor(closePressed), color.NRGBA{255, 255, 255, 255}
			case b == ButtonClose && click.Hovered():
				bg, fg = properties.CalcColor(closeHover), color.NRGBA{255, 255, 255, 255}
			case click.Pressed():
				bg = properties.CalcColor(kit.Alpha(glyphColor, 0.2))
			case click.Hovered():
				bg = properties.CalcColor(kit.Alpha(glyphColor, 0.1))
			}
			if bg.A > 0 {
				paint.FillShape(gtx.Ops, bg, clip.Rect{Max: sz}.Op())
			}
			if click.Hovered() || click.Pressed() {
				kit.MarkLive()
			}
			g := t.glyphFor(b)
			gs := gtx.Dp(10)
			goff := op.Offset(image.Pt((w-gs)/2, (h-gs)/2)).Push(gtx.Ops)
			ggtx := gtx
			ggtx.Constraints = layout.Exact(image.Pt(gs, gs))
			kit.DrawGlyph(ggtx, g, 10, fg)
			goff.Pop()
			return layout.Dimensions{Size: sz}
		})
		kit.BlockDrag(gtx, image.Pt(w, h))
		elements.RecordSemantic(gtx, elements.SemanticNode{
			Kind: elements.SemanticButton, Label: t.buttonLabel(b), Target: buttonActor{t, b},
		}, image.Pt(w, h))
		pop()
		off.Pop()
		x += w
	}
}

func (t *TitleBar) glyphFor(b Button) kit.Glyph {
	switch b {
	case ButtonMinimize:
		return kit.GlyphMinus
	case ButtonMaximize:
		if t.win.Maximized() {
			return kit.GlyphRestore
		}
		return kit.GlyphMaximize
	default:
		return kit.GlyphClose
	}
}

func (t *TitleBar) buttonLabel(b Button) string {
	switch b {
	case ButtonMinimize:
		return i18n.T("giode.minimize")
	case ButtonMaximize:
		if t.win.Maximized() {
			return i18n.T("giode.restore")
		}
		return i18n.T("giode.maximize")
	default:
		return i18n.T("giode.close")
	}
}

// --- Mac-style buttons -------------------------------------------------------

func (t *TitleBar) macButtons(focused bool) elements.Element {
	return elements.Raw(func(gtx layout.Context) layout.Dimensions {
		d := gtx.Dp(12)
		gap := gtx.Dp(8)
		order := []Button{ButtonClose, ButtonMinimize, ButtonMaximize}
		anyHover := false
		for _, b := range order {
			anyHover = anyHover || t.buttons[b].Hovered()
		}
		x := 0
		for _, b := range order {
			if t.hidden[b] {
				continue
			}
			off := op.Offset(image.Pt(x, 0)).Push(gtx.Ops)
			pop := elements.PushOffset(image.Pt(x, 0))
			click := &t.buttons[b]
			bgtx := gtx
			bgtx.Constraints = layout.Exact(image.Pt(d, d))
			click.Layout(bgtx, func(gtx layout.Context) layout.Dimensions {
				col := properties.CalcColor(macColors[b])
				if !focused && !anyHover {
					col = properties.CalcColor("theme.border")
				}
				if click.Pressed() {
					col = properties.CalcColor(kit.Pressed(properties.CalcColorReverse(col)))
				}
				paint.FillShape(gtx.Ops, col, clip.Ellipse{Max: image.Pt(d, d)}.Op(gtx.Ops))
				if anyHover {
					kit.MarkLive()
					g := macGlyphs[b]
					gs := gtx.Dp(8)
					goff := op.Offset(image.Pt((d-gs)/2, (d-gs)/2)).Push(gtx.Ops)
					ggtx := gtx
					ggtx.Constraints = layout.Exact(image.Pt(gs, gs))
					kit.DrawGlyph(ggtx, g, 8, color.NRGBA{A: 0x99})
					goff.Pop()
				}
				if click.Hovered() {
					pointer.CursorPointer.Add(gtx.Ops)
				}
				return layout.Dimensions{Size: image.Pt(d, d)}
			})
			kit.BlockDrag(gtx, image.Pt(d, d))
			elements.RecordSemantic(gtx, elements.SemanticNode{
				Kind: elements.SemanticButton, Label: t.buttonLabel(b), Target: buttonActor{t, b},
			}, image.Pt(d, d))
			pop()
			off.Pop()
			x += d + gap
		}
		if x > 0 {
			x -= gap
		}
		return layout.Dimensions{Size: image.Pt(x, d)}
	})
}

// buttonActor lets automation (the MCP server) press a window button.
type buttonActor struct {
	t *TitleBar
	b Button
}

func (a buttonActor) Click() { a.t.Press(a.b) }
