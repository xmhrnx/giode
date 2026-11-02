package tabs

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

// Layout lays out the tab bar and the active view below it.
func (t *Tabs) Layout(gtx layout.Context) layout.Dimensions {
	if len(t.tabs) == 0 {
		return layout.Dimensions{}
	}
	active := t.st.Color
	if active == "" {
		active = defaultActive
	}
	activeText := t.st.BorderColor
	if activeText == "" {
		activeText = defaultActiveText
	}

	var size image.Point
	dims := elements.Row(styles.Styles{Gap: 4}, t.tabButtons(active, activeText)...).Layout(gtx)
	size.Y += dims.Size.Y

	if t.selected < len(t.tabs) {
		gtx.Constraints.Min.Y -= dims.Size.Y
		gtx.Constraints.Max.Y -= dims.Size.Y
		off := op.Offset(image.Pt(0, dims.Size.Y)).Push(gtx.Ops)
		pop := elements.PushOffset(image.Pt(0, dims.Size.Y))
		contentDims := t.tabs[t.selected].View().Layout(gtx)
		pop()
		off.Pop()
		size.Y += contentDims.Size.Y
		if contentDims.Size.X > size.X {
			size.X = contentDims.Size.X
		}
	}
	return layout.Dimensions{Size: size}
}

// tabButtons returns one header element per tab.
func (t *Tabs) tabButtons(active, activeText properties.Color) []elements.Element {
	els := make([]elements.Element, 0, len(t.tabs))
	for i := range t.tabs {
		els = append(els, t.tabButton(i, active, activeText))
	}
	return els
}

// tabButton renders one clickable tab header.
func (t *Tabs) tabButton(i int, active, activeText properties.Color) elements.Element {
	click := t.clicks[i]
	update := func(gtx layout.Context) bool {
		if click.Clicked(gtx) {
			t.selected = i
			if t.onChange != nil {
				t.onChange(i)
			}
			kit.RequestFrame(gtx)
			return true
		}
		return false
	}
	return elements.Raw(func(gtx layout.Context) layout.Dimensions {
		update(gtx)
		kit.HandleInput(kit.ClickablePass(click, update))
		dims := click.Layout(gtx, kit.Unselectable(func(gtx layout.Context) layout.Dimensions {
			sel := i == t.selected
			textColor := defaultText
			if sel {
				textColor = activeText
			} else if click.Hovered() {
				textColor = kit.Hovered(defaultText)
			}
			if click.Hovered() {
				pointer.CursorPointer.Add(gtx.Ops)
			}
			st := styles.Styles{
				Padding: properties.SymmetricInset(12, 8),
			}
			content := elements.Text(t.tabs[i].Label, styles.Styles{Color: textColor, FontSize: 14})
			if !sel {
				return elements.Box(st, content).Layout(gtx)
			}
			// Selected tab: label + 2px underline.
			macro := op.Record(gtx.Ops)
			dims := elements.Box(st, content).Layout(gtx)
			contentOps := macro.Stop()
			defer clip.Rect{Max: dims.Size}.Push(gtx.Ops).Pop()
			contentOps.Add(gtx.Ops)
			line := gtx.Dp(2)
			off := op.Offset(image.Pt(0, dims.Size.Y-line)).Push(gtx.Ops)
			paint.FillShape(gtx.Ops, properties.CalcColor(active), clip.Rect{Max: image.Pt(dims.Size.X, line)}.Op())
			off.Pop()
			return dims
		}))
		// Keep the tab clickable even inside a frameless window's
		// whole-window drag area - see button.Layout's identical
		// comment.
		kit.BlockDrag(gtx, dims.Size)
		value := ""
		if i == t.selected {
			value = "selected"
		}
		elements.RecordSemantic(gtx, elements.SemanticNode{Kind: elements.SemanticTab, Label: t.tabs[i].Label, Value: value, Target: tabActor{t, i}}, dims.Size)
		return dims
	})
}

// tabActor lets automation select a tab as a click would.
type tabActor struct {
	t *Tabs
	i int
}

func (a tabActor) Click() {
	a.t.selected = a.i
	if a.t.onChange != nil {
		a.t.onChange(a.i)
	}
}
