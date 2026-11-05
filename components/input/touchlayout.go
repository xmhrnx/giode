package input

import (
	"image"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/i18n"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// layout registers the long-press area over the editor, size dims, and
// the menu and handles while they show. Call it right after laying the
// editor out.
func (m *editMenu) layout(gtx layout.Context, in *Input, size image.Point) {
	elements.TrackPosition(&m.origin)
	ps := pointer.PassOp{}.Push(gtx.Ops)
	cl := clip.Rect{Max: size}.Push(gtx.Ops)
	event.Op(gtx.Ops, &m.tag)
	cl.Pop()
	ps.Pop()
	if !m.open {
		return
	}
	ed := &in.editor
	acts := m.actions(in)
	s, e := ed.Selection()
	lo, hi := min(s, e), max(s, e)
	// The selection's (or caret's) rectangles, in window coordinates.
	origin := m.origin
	var bounds image.Rectangle
	var startPt, endPt image.Point
	var have bool
	regions := ed.Regions(lo, hi, nil)
	for i, r := range regions {
		b := r.Bounds.Add(origin)
		if i == 0 {
			bounds = b
			startPt = image.Pt(b.Min.X, b.Max.Y)
		} else {
			bounds = bounds.Union(b)
		}
		endPt = image.Pt(b.Max.X, b.Max.Y)
		have = true
	}
	if !have {
		c := ed.CaretCoords().Round().Add(origin)
		lh := gtx.Sp(properties.ResolveFontSize(in.st.FontSize))
		bounds = image.Rect(c.X, c.Y-lh, c.X+1, c.Y+lh/4)
		startPt, endPt = image.Pt(c.X, bounds.Max.Y), image.Pt(c.X, bounds.Max.Y)
	}
	showHandles := m.handles && lo != hi && len(regions) > 0
	dragging := m.start.drag.Dragging() || m.end.drag.Dragging()
	if m.buttons == nil {
		m.buttons = map[string]*elements.Pressable{}
	}
	elements.RegisterOverlay(func() elements.Element {
		return elements.Raw(func(gtx layout.Context) layout.Dimensions {
			win := gtx.Constraints.Max
			// Watch presses everywhere without taking them.
			ps := pointer.PassOp{}.Push(gtx.Ops)
			cl := clip.Rect{Max: win}.Push(gtx.Ops)
			event.Op(gtx.Ops, &m.outside)
			cl.Pop()
			ps.Pop()

			if showHandles {
				m.start.at, m.end.at = startPt, endPt
				m.layoutHandle(gtx, &m.start)
				m.layoutHandle(gtx, &m.end)
			}
			if len(acts) > 0 && !dragging {
				m.layoutBar(gtx, acts, bounds, showHandles)
			}
			if m.action != "" {
				m.do(gtx, in, m.action)
				m.action = ""
			}
			return layout.Dimensions{Size: win}
		})
	})
}

// layoutBar draws the action bar above the selection, or below it when
// there's no room above.
func (m *editMenu) layoutBar(gtx layout.Context, acts []string, sel image.Rectangle, handles bool) {
	win := gtx.Constraints.Max
	items := make([]elements.Element, 0, len(acts))
	for _, a := range acts {
		btn, ok := m.buttons[a]
		if !ok {
			action := a
			btn = elements.NewPressable(func(p elements.PressState) elements.Element {
				bg := ""
				if p.Pressed || p.Hovered {
					bg = "#374151"
				}
				return elements.Box(styles.Styles{Background: properties.Color(bg), BorderRadius: 6, Padding: properties.SymmetricInset(12, 8)},
					elements.Text(i18n.T("giode."+action), styles.Styles{Color: "#f9fafb", FontSize: 14}))
			}).Label(i18n.T("giode." + a)).OnClick(func(elements.ClickEvent) { m.action = action })
			m.buttons[a] = btn
		}
		items = append(items, btn)
	}
	bar := elements.Box(styles.Styles{Background: "#1f2937", BorderRadius: 8, Padding: properties.UniformInset(2),
		BoxShadow: properties.BoxShadow{Y: 2, Blur: 8, Color: "#00000040"}}, items...)

	track := elements.TrackChild()
	macro := op.Record(gtx.Ops)
	cgtx := gtx
	cgtx.Constraints.Min = image.Point{}
	dims := bar.Layout(cgtx)
	call := macro.Stop()
	track.End()

	gap := gtx.Dp(8)
	pos := image.Pt((sel.Min.X+sel.Max.X)/2-dims.Size.X/2, sel.Min.Y-dims.Size.Y-gap)
	if pos.Y < gap {
		below := sel.Max.Y + gap
		if handles {
			below += gtx.Dp(30) // clear of the handles
		}
		pos.Y = below
	}
	pos.X = max(gap, min(pos.X, win.X-dims.Size.X-gap))
	track.Place(pos)
	defer op.Offset(pos).Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
}
