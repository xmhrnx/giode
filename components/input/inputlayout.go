package input

import (
	"image"

	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Layout lays out and updates the field.
func (i *Input) Layout(gtx layout.Context) layout.Dimensions {
	dims := i.layoutSelf(gtx)
	// A right-click or long-press anywhere on it opens its context menu.
	i.AnchorContext(gtx, dims.Size)
	return dims
}

// layoutSelf lays it out without its context menu area.
func (i *Input) layoutSelf(gtx layout.Context) layout.Dimensions {
	if i.requestFocus {
		i.requestFocus = false
		gtx.Execute(key.FocusCmd{Tag: &i.editor})
	}
	i.update(gtx)
	if i.pass == nil {
		i.pass = i.update
	}
	kit.HandleInput(i.pass)
	if i.focused {
		// A focused field receives key events and blinks its caret, so
		// cached ancestors must keep laying it out.
		kit.MarkLive()
		// A single-line field's return key reads Done (it submits and
		// closes the keyboard); a multi-line one's makes a new line.
		if kit.SetReturnKeyDone != nil {
			kit.SetReturnKeyDone(i.editor.SingleLine && i.editor.Submit)
		}
	}
	// Where the field is, for a tap outside it to dismiss the keyboard.
	elements.TrackPosition(&i.fieldOrigin)

	st := styles.Merge(styles.Styles{
		Padding:      properties.SymmetricInset(12, 8),
		BorderWidth:  1,
		BorderRadius: 6,
		BorderColor:  "theme.border",
		Color:        "theme.text",
	}, i.st)
	if st.Width == 0 {
		st.Width = -1
	}

	dims := elements.Box(st, &editorEl{input: i, st: st}).Layout(gtx)
	i.fieldSize = dims.Size
	if i.focused && gtx.Source.Enabled() {
		elements.SetKeyboardOwner(i)
	}
	// A press anywhere on the field focuses it, not only on the text:
	// an empty or multiline field is mostly padding and blank space,
	// which on a touch screen otherwise makes it feel unresponsive.
	// The area passes presses through so the editor beneath still
	// places the caret and starts selections.
	pass := pointer.PassOp{}.Push(gtx.Ops)
	area := clip.Rect{Max: dims.Size}.Push(gtx.Ops)
	event.Op(gtx.Ops, &i.tap)
	area.Pop()
	pass.Pop()
	label := i.label
	if label == "" {
		label = i.placeholder
	}
	value := i.editor.Text()
	if i.editor.Mask != 0 {
		value = ""
	}
	elements.RecordSemantic(gtx, elements.SemanticNode{Kind: elements.SemanticInput, Label: label, Value: value, Target: i}, dims.Size)
	return dims
}

// KeepsKeyboard reports whether a tap at p, in window coordinates,
// lands on the field, or anywhere while its edit menu shows (its menu
// and handles sit outside it; a tap elsewhere closes the menu first).
func (i *Input) KeepsKeyboard(p image.Point) bool {
	if i.menu.open {
		return true
	}
	return p.In(image.Rectangle{Min: i.fieldOrigin, Max: i.fieldOrigin.Add(i.fieldSize)})
}
