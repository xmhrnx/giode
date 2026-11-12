package input

import (
	"image"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
)

// Input is a stateful text field, single-line by default. Create it
// once with New and lay it out every frame.
type Input struct {
	// Contextual gives it a context menu (Context).
	elements.Contextual

	pass         func(gtx layout.Context) bool
	editor       widget.Editor
	placeholder  string
	label        string
	keyboard     Keyboard
	st           styles.Styles
	onSubmit     func(text string)
	onChange     func(text string)
	requestFocus bool
	focused      bool
	submitOn     SubmitMode
	minLines     int
	maxLines     int
	onFocus      func()
	onBlur       func()
	onPaste      func(pasteText func())
	// pastePending is set by OnPaste's pasteText: the clipboard's text
	// is read in the next update.
	pastePending bool
	// tap is the event tag for presses anywhere on the field, padding
	// and border included, which focus the editor.
	tap int
	// menu is the long-press (and right-click) editing menu.
	menu editMenu
	// fieldOrigin and fieldSize are where the field was laid out, in
	// window coordinates once the frame's layout is done.
	fieldOrigin, fieldSize image.Point
}

// New returns an Input with the given placeholder. The styles
// argument is optional.
func New(placeholder string, st ...styles.Styles) *Input {
	i := &Input{placeholder: placeholder}
	if len(st) > 0 {
		i.st = st[0]
	}
	i.editor.SingleLine = true
	i.editor.Submit = true
	return i
}

// Text returns the current content.
func (i *Input) Text() string {
	return i.editor.Text()
}

// SetText replaces the content without invoking OnChange, and leaves
// the caret at the end, as typing it would: a recalled history line or
// a restored draft continues where it ends. SetCaret places it
// elsewhere.
func (i *Input) SetText(s string) {
	i.editor.SetText(s)
	n := i.editor.Len()
	i.editor.SetCaret(n, n)
}

// Submit sets the handler invoked when Enter is pressed in a
// single-line field. It returns i for chaining.
func (i *Input) Submit(fn func(text string)) *Input {
	i.onSubmit = fn
	return i
}

// OnChange sets the handler invoked after every edit. It returns i for
// chaining.
func (i *Input) OnChange(fn func(text string)) *Input {
	i.onChange = fn
	return i
}

// Password masks the content with bullets and asks touch devices for
// the password keyboard. It returns i for chaining.
func (i *Input) Password(on bool) *Input {
	if on {
		i.editor.Mask = '•'
	} else {
		i.editor.Mask = 0
	}
	i.updateHint()
	return i
}

// Keyboard selects the on-screen keyboard shown while the field is
// focused on a touch device; Password overrides it with the password
// keyboard. It returns i for chaining.
func (i *Input) Keyboard(k Keyboard) *Input {
	i.keyboard = k
	i.updateHint()
	return i
}

func (i *Input) updateHint() {
	if i.editor.Mask != 0 {
		i.editor.InputHint = key.HintPassword
	} else {
		i.editor.InputHint = i.keyboard.hint()
	}
}

// Multiline lets the field hold several lines; Enter inserts a newline
// instead of submitting (see SubmitOn). It returns i for chaining.
func (i *Input) Multiline(on bool) *Input {
	i.editor.SingleLine = !on
	i.editor.Submit = !on
	return i
}

// SubmitOn selects which key presses call the Submit handler; see
// SubmitMode. It returns i for chaining.
func (i *Input) SubmitOn(m SubmitMode) *Input {
	i.submitOn = m
	return i
}

// Lines makes a multiline field grow with its content from min to max
// lines, then scroll; 0 for max means no limit. It also turns
// Multiline on. It returns i for chaining.
func (i *Input) Lines(min, max int) *Input {
	i.minLines, i.maxLines = min, max
	return i.Multiline(true)
}

// OnFocus sets a handler called when the field gains keyboard focus.
// It returns i for chaining.
func (i *Input) OnFocus(fn func()) *Input {
	i.onFocus = fn
	return i
}

// OnBlur sets a handler called when the field loses keyboard focus.
// It returns i for chaining.
func (i *Input) OnBlur(fn func()) *Input {
	i.onBlur = fn
	return i
}

// OnPaste sets fn to run in place of pasting when the user pastes into
// the field: Ctrl+V (Cmd+V on Apple platforms) or Paste in the editing
// menu. fn runs on the UI goroutine; it may call pasteText to paste
// the clipboard's text as the field would have, then or later from the
// UI goroutine (through App.Post, from a goroutine). A chat composer
// attaches a copied image this way and pastes text otherwise:
//
//	in.OnPaste(func(pasteText func()) {
//		go func() {
//			img, err := app.ReadClipboardImage(ctx)
//			app.Post(func() {
//				if err == nil && img != nil {
//					attach(img)
//				} else {
//					pasteText()
//				}
//			})
//		}()
//	})
//
// A read-only field doesn't paste and doesn't call fn. It returns i
// for chaining.
func (i *Input) OnPaste(fn func(pasteText func())) *Input {
	i.onPaste = fn
	return i
}

// Placeholder replaces the text shown while the field is empty. It
// returns i for chaining.
func (i *Input) Placeholder(s string) *Input {
	i.placeholder = s
	return i
}

// Styles replaces the field's styles, from the next frame, keeping its
// text and focus. It returns i for chaining.
func (i *Input) Styles(st styles.Styles) *Input {
	i.st = st
	return i
}

// ReadOnly makes the content selectable but not editable. It returns i
// for chaining.
func (i *Input) ReadOnly(on bool) *Input {
	i.editor.ReadOnly = on
	return i
}

// MaxLength limits the content to n runes; 0 removes the limit. It
// returns i for chaining.
func (i *Input) MaxLength(n int) *Input {
	i.editor.MaxLen = n
	return i
}

// Label sets the accessible name reported to automation (the semantic
// tree, the MCP server); the placeholder is used when unset. It
// returns i for chaining.
func (i *Input) Label(label string) *Input {
	i.label = label
	return i
}

// Focus requests keyboard focus.
func (i *Input) Focus() {
	i.requestFocus = true
}

// Focused reports whether the field had keyboard focus in the last
// frame.
func (i *Input) Focused() bool {
	return i.focused
}
