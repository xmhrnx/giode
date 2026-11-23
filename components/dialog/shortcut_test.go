package dialog_test

import (
	"image"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/components/dialog"
	textinput "github.com/xmhrnx/giode/components/input"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/ui"
)

// A dialog opened by a shortcut while a text field has focus takes its
// first click: the shortcut asks for the frame that shows it.
func TestDialogFromShortcutTakesFirstClick(t *testing.T) {
	for _, kind := range []string{"input", "keyboard", "input-nofocus", "keyboard-nofocus"} {
		t.Run(kind, func(t *testing.T) { testDialogFromShortcut(t, kind) })
	}
}

func testDialogFromShortcut(t *testing.T, kind string) {
	var r input.Router
	open := false
	var field interface {
		elements.Element
		Focus()
		Focused() bool
	}
	if kind == "input" || kind == "input-nofocus" {
		field = textinput.New("Search")
	} else {
		field = textinput.NewKeyboardInput(elements.Raw(func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: image.Pt(600, 400)}
		}))
	}
	clicks := 0
	target := elements.NewPressable(func(elements.PressState) elements.Element {
		return elements.Raw(func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: image.Pt(200, 100)}
		})
	}).OnClick(func(elements.ClickEvent) { clicks++ })
	if kind == "input-nofocus" || kind == "keyboard-nofocus" {
		target.Focusable(false)
	}
	d := dialog.New(&open, func() elements.Element { return target })
	now := time.Unix(0, 0)
	frame := func() {
		now = now.Add(16 * time.Millisecond)
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Now: now, Source: r.Source(), Constraints: layout.Exact(image.Pt(600, 400))}
		ui.Frame(gtx, elements.Raw(func(gtx layout.Context) layout.Dimensions {
			elements.OnKey(func(e key.Event) bool {
				if e.Name == "K" && e.State == key.Press {
					open = true
					return true
				}
				return false
			}, elements.NamedKey{Name: "K", Mods: key.ModCtrl})
			return elements.Stack(styles.Styles{}, field, d).Layout(gtx)
		}))
		r.Frame(&ops)
	}
	frame()
	field.Focus()
	frame()
	frame()
	if !field.Focused() {
		t.Fatal("setup: field not focused")
	}
	r.Queue(key.Event{Name: "K", Modifiers: key.ModCtrl, State: key.Press})
	frame()
	// The shortcut ran after the frame was laid out: the frame showing
	// the dialog must follow at once, not with the next input - else
	// the click meant for the dialog lands on the frame without it.
	if _, ok := r.WakeupTime(); !ok {
		t.Fatal("the shortcut didn't ask for a frame")
	}
	r.Queue(key.Event{Name: "K", Modifiers: key.ModCtrl, State: key.Release})
	for range 3 {
		frame()
	}
	if !open {
		t.Fatal("the shortcut didn't open the dialog")
	}
	at := f32.Pt(300, 200)
	r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: at})
	r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: at, Buttons: pointer.ButtonPrimary})
	frame()
	r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: at})
	frame()
	frame()
	if clicks != 1 {
		t.Errorf("the first click in the dialog: %d clicks, want 1", clicks)
	}
}
