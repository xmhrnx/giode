// Package segmented provides a segmented control: a row of mutually
// exclusive options in one rounded track, as iOS and Android use for
// switching views or modes.

package segmented

import (
	"image"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Layout lays out and updates the control.
func (c *Control) Layout(gtx layout.Context) layout.Dimensions {
	dims := c.layoutSelf(gtx)
	// A right-click or long-press anywhere on it opens its context menu.
	c.AnchorContext(gtx, dims.Size)
	return dims
}

// layoutSelf lays it out without its context menu area.
func (c *Control) layoutSelf(gtx layout.Context) layout.Dimensions {
	n := len(c.options)
	if n == 0 {
		return layout.Dimensions{}
	}
	c.update(gtx)
	if c.pass == nil {
		c.pass = func(gtx layout.Context) bool {
			var hov [16]bool
			for i := range c.clicks {
				if i < len(hov) {
					hov[i] = c.clicks[i].Hovered()
				}
			}
			changed := c.update(gtx)
			for i := range c.clicks {
				if i < len(hov) && hov[i] != c.clicks[i].Hovered() {
					changed = true
				}
			}
			return changed
		}
	}
	kit.HandleInput(c.pass)
	dt := gtx.Now.Sub(c.last)
	if c.last.IsZero() || dt < 0 || dt > animDuration {
		dt = animDuration
	}
	c.last = gtx.Now
	if target := float32(c.selected); c.pos != target {
		step := float32(dt) / float32(animDuration) * max(1, abs(target-c.pos))
		if c.pos < target {
			c.pos = min(target, c.pos+step)
		} else {
			c.pos = max(target, c.pos-step)
		}
		kit.RequestFrame(gtx)
	}

	st := styles.Merge(styles.Styles{Background: trackColor, BorderColor: "theme.surface", Color: "theme.text", FontSize: 14}, c.st)
	pad := gtx.Dp(3)
	segment := func(i int, fill bool) elements.Element {
		var kids []elements.Element
		if i < len(c.icons) && c.icons[i] != nil {
			kids = append(kids, c.icons[i])
		}
		if c.options[i] != "" {
			w := properties.FontWeightNormal
			if i == c.selected {
				w = properties.FontWeightSemiBold
			}
			kids = append(kids, elements.Text(c.options[i], styles.Styles{Color: st.Color, FontSize: st.FontSize, FontWeight: w}))
		}
		var w properties.Width
		if fill {
			w = -1
		}
		return elements.Row(styles.Styles{Width: w, Justify: properties.JustifyCenter, Align: properties.AlignCenter, Gap: 6, Padding: properties.SymmetricInset(14, 7), Color: st.Color}, kids...)
	}

	// Segments share the width equally: the widest one's natural
	// width, or the whole row with Width -1.
	mgtx := gtx
	mgtx.Constraints.Min = image.Point{}
	segW, segH := 0, 0
	for i := range c.options {
		d := elements.Measure(mgtx, segment(i, false))
		segW, segH = max(segW, d.Size.X), max(segH, d.Size.Y)
	}
	switch w := c.st.Px(gtx.Metric).Width; {
	case w == -1:
		segW = (gtx.Constraints.Max.X - 2*pad) / n
	case w > 0:
		segW = max(0, min(int(w), gtx.Constraints.Max.X)-2*pad) / n
	}
	segH = max(segH, gtx.Dp(32))
	size := image.Pt(segW*n+2*pad, segH+2*pad)
	radius := size.Y / 2

	func() {
		defer clip.UniformRRect(image.Rectangle{Max: size}, radius).Push(gtx.Ops).Pop()
		paint.Fill(gtx.Ops, properties.CalcColor(st.Background))
	}()
	// Right to left, the first option is the rightmost.
	rtl := kit.RTL(gtx)
	hpos := c.pos
	if rtl {
		hpos = float32(n-1) - c.pos
	}
	hx := pad + int(hpos*float32(segW))
	func() {
		defer clip.UniformRRect(image.Rect(hx, pad, hx+segW, pad+segH), radius-pad).Push(gtx.Ops).Pop()
		paint.Fill(gtx.Ops, properties.CalcColor(st.BorderColor))
	}()
	for i := range c.options {
		func() {
			slot := i
			if rtl {
				slot = n - 1 - i
			}
			pos := image.Pt(pad+slot*segW, pad)
			defer op.Offset(pos).Push(gtx.Ops).Pop()
			pop := elements.PushOffset(pos)
			defer pop()
			sgtx := gtx
			sgtx.Constraints = layout.Exact(image.Pt(segW, segH))
			d := c.clicks[i].Layout(sgtx, kit.Unselectable(func(gtx layout.Context) layout.Dimensions {
				if c.clicks[i].Hovered() {
					pointer.CursorPointer.Add(gtx.Ops)
				}
				return elements.Box(styles.Styles{Width: -1, Height: -1, Justify: properties.JustifyCenter, Align: properties.AlignCenter}, segment(i, true)).Layout(gtx)
			}))
			elements.RecordSemantic(gtx, elements.SemanticNode{Kind: elements.SemanticTab, Label: c.options[i], Value: boolString(i == c.selected), Target: &c.clicks[i]}, d.Size)
		}()
	}
	kit.BlockDrag(gtx, size)
	elements.RecordSemantic(gtx, elements.SemanticNode{Kind: elements.SemanticSelect, Label: c.options[c.selected], Value: c.options[c.selected], Target: c}, size)
	return layout.Dimensions{Size: size}
}
