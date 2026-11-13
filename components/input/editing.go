package input

import (
	"gioui.org/io/clipboard"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/widget"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"runtime"
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

// paste pastes the clipboard's text, or hands the paste to OnPaste.
func (i *Input) paste(gtx layout.Context) {
	if i.editor.ReadOnly {
		return
	}
	if i.onPaste == nil {
		gtx.Execute(clipboard.ReadCmd{Tag: &i.editor})
		return
	}
	i.onPaste(func() {
		i.pastePending = true
	})
	if i.pastePending {
		i.pastePending = false
		gtx.Execute(clipboard.ReadCmd{Tag: &i.editor})
	}
	kit.RequestFrame(gtx)
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

// Caret returns the caret's position: the number of runes before it.
// With a selection, it's the end the caret is at (see Selection).
func (i *Input) Caret() int {
	c, _ := i.editor.Selection()
	return c
}

// SetCaret moves the caret to pos runes from the start, clamped to the
// content, and clears the selection.
func (i *Input) SetCaret(pos int) {
	pos = max(0, min(pos, i.editor.Len()))
	i.editor.SetCaret(pos, pos)
}

// Selection returns the selected runes as [start, end), start <= end;
// start == end is the caret alone.
func (i *Input) Selection() (start, end int) {
	a, b := i.editor.Selection()
	return min(a, b), max(a, b)
}

// SetSelection selects the runes [start, end), clamped to the content,
// with the caret at end; start == end places the caret alone. To
// replace the word at the caret, select it and call InsertText.
func (i *Input) SetSelection(start, end int) {
	n := i.editor.Len()
	start, end = max(0, min(start, n)), max(0, min(end, n))
	i.editor.SetCaret(end, start)
}

// ChangeText replaces the content and invokes OnChange, as typing
// would. Like SetText, it leaves the caret at the end.
func (i *Input) ChangeText(s string) {
	i.editor.SetText(s)
	n := i.editor.Len()
	i.editor.SetCaret(n, n)
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
	// Focus and Blur apply here, in a real pass (a measuring pass's
	// commands are dropped), and also while a cached ancestor replays.
	if gtx.Source.Enabled() {
		if i.requestFocus {
			i.requestFocus = false
			gtx.Execute(key.FocusCmd{Tag: &i.editor})
			changed = true
		}
		if i.requestBlur {
			i.requestBlur = false
			if gtx.Source.Focused(&i.editor) {
				gtx.Execute(key.FocusCmd{})
				gtx.Execute(key.SoftKeyboardCmd{Show: false})
				changed = true
			}
		}
	}
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
	for _, n := range i.keys {
		for {
			ev, ok := gtx.Event(key.Filter{Focus: &i.editor, Name: n, Optional: key.ModShift | key.ModCtrl | key.ModAlt | key.ModCommand})
			if !ok {
				break
			}
			if e, ok := ev.(key.Event); ok && e.State == key.Press && i.onKeys != nil {
				i.onKeys(e)
				changed = true
			}
		}
	}
	if i.onPaste != nil {
		// Ctrl+V is taken before the editor sees it, so OnPaste can
		// decide what's pasted.
		for {
			ev, ok := gtx.Event(key.Filter{Focus: &i.editor, Name: "V", Required: key.ModShortcut})
			if !ok {
				break
			}
			if e, ok := ev.(key.Event); ok && e.State == key.Press {
				i.paste(gtx)
			}
		}
	}
	if i.pastePending {
		i.pastePending = false
		if !i.editor.ReadOnly {
			// The editor inserts the text when it arrives.
			gtx.Execute(clipboard.ReadCmd{Tag: &i.editor})
			changed = true
		}
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
			if i.editor.SingleLine && phone {
				// Return (Done) on a single-line field closes the
				// keyboard, as on native fields.
				gtx.Execute(key.FocusCmd{})
				gtx.Execute(key.SoftKeyboardCmd{Show: false})
			}
		case widget.ChangeEvent:
			if i.onChange != nil {
				i.onChange(i.editor.Text())
				kit.RequestFrame(gtx)
			}
		}
	}
	if !gtx.Source.Enabled() {
		// A measuring pass: Focused reports false whatever the focus,
		// which isn't the field losing it.
		return changed
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

// phone is set on Android and iOS, whose keyboards close with Return on
// single-line fields.
const phone = runtime.GOOS == "android" || runtime.GOOS == "ios"
