// Package titlebar provides a custom window title bar - drag handle,
// title, optional icon and menus, and minimize/maximize/close buttons
// - for frameless windows that replace the operating system's own.

package titlebar

import (
	"image"
	"image/color"
	"slices"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/i18n"
	"github.com/xmhrnx/giode/styles/properties"
)

// buttonActor lets automation (the MCP server) press a window button.
type buttonActor struct {
	t *TitleBar
	b Button
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

func (a buttonActor) Click() { a.t.Press(a.b) }
