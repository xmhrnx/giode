package giode

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/ui"
)

// Where an app has one window, a portal covers the main window: it
// takes the presses meant for the view beneath, shows its title bar,
// and Back closes it.
func TestInlinePortal(t *testing.T) {
	old := singleWindow
	singleWindow = func() bool { return true }
	defer func() { singleWindow = old }()

	a := &App{cfg: Config{Background: "#ffffff"}}
	laidOut := 0
	closed := false
	w := a.OpenPortal(PortalOptions{Title: "Inspector", OnClose: func() { closed = true }}, func() Element {
		return elements.Raw(func(gtx layout.Context) layout.Dimensions {
			laidOut++
			return layout.Dimensions{Size: gtx.Constraints.Max}
		})
	})
	if got := a.Portals(); len(got) != 1 || got[0] != w || !w.inline {
		t.Fatalf("Portals() = %v", got)
	}

	var (
		r       input.Router
		ops     op.Ops
		beneath = new(int)
		pressed bool
	)
	frame := func() {
		ops.Reset()
		gtx := layout.Context{Ops: &ops, Source: r.Source(), Constraints: layout.Exact(image.Pt(300, 400))}
		ui.Frame(gtx, elements.Raw(func(gtx layout.Context) layout.Dimensions {
			for {
				ev, ok := gtx.Event(pointer.Filter{Target: beneath, Kinds: pointer.Press})
				if !ok {
					break
				}
				if _, ok := ev.(pointer.Event); ok {
					pressed = true
				}
			}
			cl := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
			event.Op(gtx.Ops, beneath)
			cl.Pop()
			a.layoutInlinePortals(gtx)
			return layout.Dimensions{Size: gtx.Constraints.Max}
		}))
		r.Frame(&ops)
	}
	frame()
	if laidOut == 0 {
		t.Fatal("the portal's view wasn't laid out")
	}
	r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: f32.Pt(150, 300)},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: f32.Pt(150, 300)})
	frame()
	if pressed {
		t.Error("a press reached the view beneath the portal")
	}

	r.Queue(key.Event{Name: key.NameBack, State: key.Press})
	frame()
	if !w.Closed() || len(a.Portals()) != 0 {
		t.Fatalf("Back didn't close the portal: closed %v, portals %v", w.Closed(), a.Portals())
	}
	a.runUICalls()
	if !closed {
		t.Error("OnClose didn't run")
	}
}
