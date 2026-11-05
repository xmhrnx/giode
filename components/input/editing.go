package input

import (
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/widget"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
)

// SelectAll selects the whole content.
func (i *Input) SelectAll() {
	i.editor.SetCaret(i.editor.Len(), 0)
}

// InsertText replaces the selection (or inserts at the caret) with s
// and invokes OnChange, as typing would.
func (i *Input) InsertText(s string) {
	i.editor.Insert(s)
	if i.onChange != nil {
		i.onChange(i.editor.Text())
	}
}

// applySubmit sets the editor's submit behavior for this frame.
func (i *Input) applySubmit() {
	switch i.submitOn {
	case SubmitEnter:
		i.editor.Submit = true
	case SubmitEnterUnlessShift:
		i.editor.Submit = !elements.TouchInput()
	case SubmitNever:
		i.editor.Submit = false
	default:
		i.editor.Submit = i.editor.SingleLine
	}
}

// ChangeText replaces the content and invokes OnChange, as typing
// would.
func (i *Input) ChangeText(s string) {
	i.editor.SetText(s)
	if i.onChange != nil {
		i.onChange(s)
	}
}

// SubmitText invokes the Submit handler with the current content, as
// pressing Enter would.
func (i *Input) SubmitText() {
	if i.onSubmit != nil {
		i.onSubmit(i.editor.Text())
	}
}

// update handles taps, editing and focus, reporting whether anything
// happened that changes how the field looks.
func (i *Input) update(gtx layout.Context) (changed bool) {
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &i.tap, Kinds: pointer.Press})
		if !ok {
			break
		}
		if e, ok := ev.(pointer.Event); ok && (e.Source == pointer.Touch || e.Buttons.Contain(pointer.ButtonPrimary)) {
			changed = true
			gtx.Execute(key.FocusCmd{Tag: &i.editor})
			if !i.editor.ReadOnly {
				gtx.Execute(key.SoftKeyboardCmd{Show: true})
			}
		}
	}
	if i.menu.update(gtx, i) {
		changed = true
	}
	i.applySubmit()
	for {
		ev, ok := i.editor.Update(gtx)
		if !ok {
			break
		}
		changed = true
		switch e := ev.(type) {
		case widget.SubmitEvent:
			if i.onSubmit != nil {
				i.onSubmit(e.Text)
				kit.RequestFrame(gtx)
			}
		case widget.ChangeEvent:
			if i.onChange != nil {
				i.onChange(i.editor.Text())
				kit.RequestFrame(gtx)
			}
		}
	}
	focused := gtx.Source.Focused(&i.editor)
	if focused != i.focused {
		changed = true
		if focused && i.onFocus != nil {
			i.onFocus()
		}
		if !focused && i.onBlur != nil {
			i.onBlur()
		}
	}
	i.focused = focused
	return changed
}
