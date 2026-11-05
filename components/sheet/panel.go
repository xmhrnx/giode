// Package sheet provides a modal sheet: a panel that slides up from the
// bottom of the window on phones and, with Adaptive, shows as a
// centered dialog on wide screens.

package sheet

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// panelOps records the panel - handle, title, body, footer - at width,
// fixedH tall or, when fixedH is 0, as tall as its content up to maxH,
// and returns it with its height. Semantic nodes and portals inside
// follow it to where it's replayed.
func (s *Sheet) panelOps(gtx layout.Context, width, maxH, fixedH int, handle bool) (op.CallOp, int, elements.ChildTracker) {
	st := styles.Merge(styles.Styles{
		Background: "theme.surface",
		Color:      "theme.text",
		Padding:    properties.Inset{Left: 16, Right: 16},
	}, s.st)
	if handle {
		st.BorderRadii = properties.TopCorners(16)
	} else if st.BorderRadius == 0 {
		st.BorderRadius = 14
	}
	pad := st.Padding

	var header []elements.Element
	if handle {
		header = append(header, elements.Row(styles.Styles{Width: -1, Justify: properties.JustifyCenter, Padding: properties.Inset{Top: 8, Bottom: 8}},
			elements.Box(styles.Styles{Width: 36, Height: 4, BorderRadius: 2, Background: "theme.muted"})))
	} else {
		header = append(header, elements.Box(styles.Styles{Height: 16}))
	}
	if s.title != "" {
		header = append(header, elements.Box(styles.Styles{Padding: properties.Inset{Left: pad.Left, Right: pad.Right, Bottom: 8}},
			elements.Text(s.title, styles.Styles{Color: st.Color, FontSize: 17, FontWeight: properties.FontWeightSemiBold})))
	}
	headerEl := elements.Stack(styles.Styles{Width: -1}, header...)
	var footerEl elements.Element
	if s.footer != nil {
		footerEl = elements.Box(styles.Styles{Width: -1, Padding: properties.Inset{Top: 8, Left: pad.Left, Right: pad.Right, Bottom: 16}}, s.footer())
	} else {
		footerEl = elements.Box(styles.Styles{Height: 16})
	}

	pgtx := gtx
	pgtx.Constraints = layout.Constraints{Min: image.Pt(width, 0), Max: image.Pt(width, maxH)}
	hdrH := elements.Measure(pgtx, headerEl).Size.Y
	ftrH := elements.Measure(pgtx, footerEl).Size.Y

	bodyMax := max(0, maxH-hdrH-ftrH)
	if fixedH > 0 {
		bodyMax = max(0, fixedH-hdrH-ftrH)
	}
	body := elements.Raw(func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.Y = bodyMax
		if fixedH > 0 {
			gtx.Constraints.Min.Y = bodyMax
		}
		return s.body.Layout(gtx)
	})
	bodyEl := elements.Box(styles.Styles{Width: -1, Padding: properties.Inset{Left: pad.Left, Right: pad.Right}}, body)

	panelSt := st
	panelSt.Padding = properties.Inset{}
	panelSt.Width = -1
	panel := elements.Stack(panelSt, headerEl, bodyEl, footerEl)

	track := elements.TrackChild()
	macro := op.Record(gtx.Ops)
	dims := kit.LayoutClickable(pgtx, &s.panel, func(gtx layout.Context) layout.Dimensions {
		d := panel.Layout(gtx)
		if handle && !s.fixed {
			// The handle and title are where a drag moves the sheet.
			defer clip.Rect{Max: image.Pt(d.Size.X, hdrH)}.Push(gtx.Ops).Pop()
			properties.CursorGrab.CursorFor().Add(gtx.Ops)
			s.drag.Add(gtx.Ops)
		}
		return d
	})
	call := macro.Stop()
	track.End()
	return call, dims.Size.Y, track
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
