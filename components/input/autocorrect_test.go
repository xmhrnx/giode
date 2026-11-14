package input

import (
	"testing"

	"gioui.org/io/key"
)

// Autocorrect(false) flags the keyboard hint, keeping the layout; a
// password field stays a password field.
func TestAutocorrectHint(t *testing.T) {
	i := New("").Autocorrect(false)
	if i.editor.InputHint != key.HintAny|hintPlain {
		t.Errorf("plain text hint %d", i.editor.InputHint)
	}
	i.Keyboard(KeyboardURL)
	if i.editor.InputHint != key.HintURL|hintPlain {
		t.Errorf("plain URL hint %d", i.editor.InputHint)
	}
	i.Autocorrect(true)
	if i.editor.InputHint != key.HintURL {
		t.Errorf("URL hint %d", i.editor.InputHint)
	}
	i.Autocorrect(false).Password(true)
	if i.editor.InputHint != key.HintPassword {
		t.Errorf("password hint %d", i.editor.InputHint)
	}
}
