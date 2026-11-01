package dropdown

import (
	"image"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/unit"

	"github.com/mehran9675/giode/components/kit"
	"github.com/mehran9675/giode/elements"
	"github.com/mehran9675/giode/styles"
	"github.com/mehran9675/giode/styles/properties"
)

// Layout lays out the field and, when open, queues the dropdown panel
// as a portal (see elements.RegisterPortal) anchored just below it, so
// it paints above everything else regardless of where Layout is
// called from.
func (s *Select) Layout(gtx layout.Context) layout.Dimensions {
	// Handles its input without an input pass; see kit.OpaqueInput.
	kit.OpaqueInput()
	st := styles.Merge(styles.Styles{
		Background:   properties.CalcColorReverse(defaultBackground),
		Color:        properties.CalcColorReverse(defaultText),
		BorderWidth:  1,
		BorderColor:  properties.CalcColorReverse(defaultBorder),
		BorderRadius: 6,
		Padding:      properties.SymmetricInset(12, 8),
	}, s.st)

	if s.field.Clicked(gtx) {
		s.open = !s.open
		kit.RequestFrame(gtx)
	}
	if s.open {
		// The open panel is a portal; keep enclosing caches live so
		// it's registered every frame.
		kit.MarkLive()
	}

	// Field.
	fieldDims := kit.LayoutClickable(gtx, &s.field, kit.Unselectable(func(gtx layout.Context) layout.Dimensions {
		if s.st.Width > 0 {
			w := gtx.Dp(unit.Dp(s.st.Width))
			gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
		}
		fieldSt := st
		if s.field.Hovered() {
			fieldSt.BorderColor = kit.Hovered(st.BorderColor)
			pointer.CursorPointer.Add(gtx.Ops)
		}
		// With a fixed width the label fills the space left of the
		// chevron, pushing it to the right edge. The row must not wrap:
		// FlexGrow is ignored on wrapped lines.
		rowSt := styles.Styles{Align: properties.AlignCenter, Wrap: properties.WrapNoWrap}
		labelSt := styles.Styles{FlexGrow: 1}
		if s.st.Width > 0 {
			rowSt.Width = -1
			labelSt.Width = -1
		}
		row := elements.Row(rowSt,
			elements.Box(labelSt, elements.Text(s.Value(), styles.Styles{Color: st.Color, FontSize: st.FontSize})),
			elements.Box(styles.Styles{Width: 4}),
			elements.Glyph(kit.GlyphChevronDown, styles.Styles{Color: st.Color, Width: 16}),
		)
		return elements.Box(fieldSt, row).Layout(gtx)
	}))
	// Keep the field clickable even inside a frameless window's
	// whole-window drag area - see button.Layout's identical comment.
	kit.BlockDrag(gtx, fieldDims.Size)
	elements.RecordSemantic(gtx, elements.SemanticNode{Kind: elements.SemanticSelect, Label: s.Value(), Value: s.Value(), Target: s}, fieldDims.Size)

	if s.open && len(s.options) > 0 {
		// Back and Escape close the option list.
		elements.HandleDismiss(func() { s.open = false })
		if s.useSheet() {
			elements.RegisterOverlay(func() elements.Element {
				return elements.Raw(func(gtx layout.Context) layout.Dimensions {
					return s.layoutSheet(gtx, st)
				})
			})
			return fieldDims
		}
		// A press of any button outside the list closes it, and goes
		// no further, as for menus: the field itself included, so
		// clicking it again closes rather than reopens.
		elements.RegisterOverlay(func() elements.Element {
			return elements.Raw(s.layoutOutside)
		})
		fieldWidth := fieldDims.Size.X
		at := elements.AbsolutePosition()
		trigger := image.Rectangle{Min: at, Max: at.Add(fieldDims.Size)}
		elements.RegisterPortalAround(gtx, trigger, 4, elements.PlaceAuto, func() elements.Element {
			return elements.Raw(func(gtx layout.Context) layout.Dimensions {
				return s.dropdown(gtx, st, fieldWidth)
			})
		})
	}
	return fieldDims
}

// layoutOutside covers the window under the open list and closes it on
// a press there.
func (s *Select) layoutOutside(gtx layout.Context) layout.Dimensions {
	for {
		ev, ok := gtx.Source.Event(pointer.Filter{Target: &s.outside, Kinds: pointer.Press})
		if !ok {
			break
		}
		if _, ok := ev.(pointer.Event); ok && s.open {
			s.open = false
			kit.RequestFrame(gtx)
		}
	}
	defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, &s.outside)
	return layout.Dimensions{}
}
