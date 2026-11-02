package giode

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/components/menu"
	"github.com/xmhrnx/giode/elements"
)

func TestWindowPointer_ContextMenuOpensOverAnyWidget(t *testing.T) {
	size := image.Pt(800, 600)
	var r input.Router
	a := &App{
		cfg:     Config{Frameless: true, Draggable: true},
		ctxMenu: menu.New().Item("x", func() {}),
	}

	// A widget covering the whole window, to confirm the context menu
	// still opens even though something else claims every point (the
	// bug this test guards: it didn't, before merging the listeners
	// and moving registration after the view).
	widget := Row(Styles{Width: -1, Height: -1})

	frame := func() {
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Constraints: layout.Constraints{Max: size}, Source: r.Source()}
		layoutDrag(a, gtx)
		widget.Layout(gtx)
		a.winPointer.layout(a, gtx)
		r.Frame(&ops)
	}

	frame()
	r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: f32.Pt(400, 300), Buttons: pointer.ButtonSecondary})
	frame()
	if !a.ctxMenu.Opened() {
		t.Error("right-click over a whole-window widget: context menu did not open")
	}
}

func TestLayoutDrag_RegistersWholeWindowMove(t *testing.T) {
	size := image.Pt(800, 600)
	var r input.Router
	a := &App{cfg: Config{Frameless: true, Draggable: true}}

	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: layout.Constraints{Max: size}, Source: r.Source()}
	layoutDrag(a, gtx)
	r.Frame(&ops)

	if action, ok := r.ActionAt(f32.Pt(10, 10)); !ok || action != system.ActionMove {
		t.Errorf("ActionAt(empty point) = (%v, %v), want (ActionMove, true)", action, ok)
	}
}

func TestLayoutDrag_NotRegisteredWithoutFramelessDraggable(t *testing.T) {
	size := image.Pt(800, 600)
	var r input.Router
	a := &App{cfg: Config{Frameless: false, Draggable: true}}

	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: layout.Constraints{Max: size}, Source: r.Source()}
	layoutDrag(a, gtx)
	r.Frame(&ops)

	if action, ok := r.ActionAt(f32.Pt(10, 10)); ok {
		t.Errorf("ActionAt(empty point) = (%v, %v), want no action registered when not frameless", action, ok)
	}
}

func TestLayoutDrag_InteractiveWidgetStaysClickable(t *testing.T) {
	size := image.Pt(800, 600)
	var r input.Router
	a := &App{cfg: Config{Frameless: true, Draggable: true}}
	btn := Button("Click me")

	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: layout.Constraints{Max: size}, Source: r.Source()}
	layoutDrag(a, gtx)
	btn.Layout(gtx)
	r.Frame(&ops)

	// Inside the button, laid out at the window origin: drag must be
	// blocked so a press is delivered as a normal click, not hijacked
	// into a native window-move.
	if action, ok := r.ActionAt(f32.Pt(5, 5)); ok && action == system.ActionMove {
		t.Error("ActionAt(button) = ActionMove, want it blocked so the button stays clickable")
	}

	// Away from the button, empty space is still draggable.
	if action, ok := r.ActionAt(f32.Pt(700, 500)); !ok || action != system.ActionMove {
		t.Errorf("ActionAt(empty point) = (%v, %v), want (ActionMove, true)", action, ok)
	}
}

func TestWindowPointer_TracksTouchInput(t *testing.T) {
	defer elements.SetTouchInput(elements.TouchInput())
	elements.SetTouchInput(false)
	size := image.Pt(800, 600)
	var r input.Router
	a := &App{}
	widget := Row(Styles{Width: -1, Height: -1})
	frame := func() {
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Constraints: layout.Constraints{Max: size}, Source: r.Source()}
		widget.Layout(gtx)
		a.winPointer.layout(a, gtx)
		r.Frame(&ops)
	}
	press := func(src pointer.Source) {
		pos := f32.Pt(400, 300)
		r.Queue(pointer.Event{Kind: pointer.Press, Source: src, Position: pos, Buttons: pointer.ButtonPrimary},
			pointer.Event{Kind: pointer.Release, Source: src, Position: pos})
		frame()
	}

	frame()
	press(pointer.Touch)
	if !elements.TouchInput() {
		t.Error("touch press didn't switch to touch input")
	}
	press(pointer.Mouse)
	if elements.TouchInput() {
		t.Error("mouse press didn't switch back")
	}
}
