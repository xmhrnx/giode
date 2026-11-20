package dialog_test

import (
	"image"
	"testing"
	"time"

	"gioui.org/io/input"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/components/button"
	"github.com/xmhrnx/giode/components/dialog"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/ui"
)

func TestDialogTrapsAndRestoresFocus(t *testing.T) {
	var r input.Router
	open := false
	outside := button.New("Outside")
	ok := button.New("OK")
	cancel := button.New("Cancel")
	d := dialog.New(&open, func() elements.Element {
		return elements.Row(styles.Styles{}, ok, cancel)
	})
	now := time.Unix(0, 0)
	frame := func() {
		now = now.Add(16 * time.Millisecond)
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Now: now, Source: r.Source(), Constraints: layout.Exact(image.Pt(600, 400))}
		ui.Frame(gtx, elements.Stack(styles.Styles{}, outside, d))
		r.Frame(&ops)
	}
	focused := func(b *button.Button) bool { return b.Focused() }
	frame()
	outside.Focus()
	frame()
	frame()
	if !focused(outside) {
		t.Fatal("setup: outside button not focused")
	}
	open = true
	for range 4 {
		frame()
	}
	if !focused(ok) {
		t.Fatal("opening didn't move focus into the dialog")
	}
	// Focus escapes (as Tab past the last control wraps to the window's
	// first): it comes back.
	cancel.Focus()
	frame()
	frame()
	outside.Focus()
	for range 3 {
		frame()
	}
	if !focused(ok) {
		t.Error("focus left the open dialog")
	}
	open = false
	for range 3 {
		frame()
	}
	if !focused(outside) {
		t.Error("closing didn't give focus back")
	}
}
