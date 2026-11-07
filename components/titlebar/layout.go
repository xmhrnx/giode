// Package titlebar provides a custom window title bar - drag handle,
// title, optional icon and menus, and minimize/maximize/close buttons
// - for frameless windows that replace the operating system's own.

package titlebar

import (
	"image"

	"gioui.org/gesture"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Layout lays out the bar across the available width.
func (t *TitleBar) Layout(gtx layout.Context) layout.Dimensions {
	dims := t.layoutSelf(gtx)
	// A right-click or long-press anywhere on it opens its context menu.
	t.AnchorContext(gtx, dims.Size)
	return dims
}

// layoutSelf lays it out without its context menu area.
func (t *TitleBar) layoutSelf(gtx layout.Context) layout.Dimensions {
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

func (t *TitleBar) windowsButtonsWidth(gtx layout.Context) int {
	return len(t.visible()) * gtx.Dp(winButtonWidth)
}
