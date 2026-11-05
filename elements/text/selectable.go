package text

import (
	"image"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements/internal/base"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Selectable makes the text selectable. With a mouse, drag to select,
// double-click for a word, and press Ctrl+C (Cmd+C) to copy; a
// right-click selects it all and offers Copy. On a touch screen, a
// long-press selects it all and offers Copy, and dragging over the
// text still scrolls whatever it's in. key identifies the text across
// frames (a message ID, say), since Text itself is rebuilt every frame;
// the selection lives in the keyed store (see Keyed). It returns t for
// chaining.
func (t *TextElement) Selectable(key any) *TextElement {
	t.selKey = key
	return t
}

// layoutSelectable lays the text out with its selection state.
func (t *TextElement) layoutSelectable(gtx layout.Context, st styles.Styles, str string, face font.Font, size unit.Sp, textColor op.CallOp) layout.Dimensions {
	s := base.Keyed("giode.selectable", t.selKey, newSelectState)
	s.update(gtx)
	kit.HandleInput(s.passFunc())
	if !base.TouchInput() {
		// widget.Selectable's mouse selection handles its own events.
		kit.OpaqueInput()
	}

	selMacro := op.Record(gtx.Ops)
	paint.ColorOp{Color: properties.CalcColor(selectionColor(st.Color))}.Add(gtx.Ops)
	selColor := selMacro.Stop()

	s.sel.Alignment = st.TextAlign.LayoutAlignment()
	s.sel.MaxLines = int(st.MaxLines)
	s.sel.LineHeight = unit.Sp(st.LineHeight)
	if st.LineHeight != 0 {
		s.sel.LineHeightScale = 1
	}
	if s.sel.Text() != str {
		s.sel.SetText(str)
	}

	var dims layout.Dimensions
	if base.TouchInput() {
		// widget.Selectable's drag would take touch drags from an
		// enclosing Scroll, so on touch the text is laid out plainly
		// and selection works by long-press.
		label := widget.Label{Alignment: s.sel.Alignment, MaxLines: s.sel.MaxLines, LineHeight: s.sel.LineHeight, LineHeightScale: s.sel.LineHeightScale}
		if s.all {
			macro := op.Record(gtx.Ops)
			dims = label.Layout(gtx, fonts.Shaper(), face, size, str, textColor)
			call := macro.Stop()
			func() {
				defer clip.UniformRRect(image.Rectangle{Max: dims.Size}, gtx.Dp(3)).Push(gtx.Ops).Pop()
				paint.Fill(gtx.Ops, properties.CalcColor(selectionColor(st.Color)))
			}()
			call.Add(gtx.Ops)
		} else {
			dims = label.Layout(gtx, fonts.Shaper(), face, size, str, textColor)
		}
	} else {
		switch {
		case s.all && !s.applied:
			s.sel.SetCaret(0, len([]rune(str)))
			s.applied = true
		case !s.all && s.applied:
			s.sel.ClearSelection()
			s.applied = false
		}
		dims = s.sel.Layout(gtx, fonts.Shaper(), face, size, textColor, selColor)
		if s.applied && s.sel.SelectionLen() == 0 {
			// A click cleared the selection.
			s.all, s.applied = false, false
		}
	}
	s.layoutCopy(gtx, dims.Size, str)
	return dims
}
