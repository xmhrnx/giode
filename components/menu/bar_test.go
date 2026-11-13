package menu

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
)

func TestBar_SwitchesMenus(t *testing.T) {
	elements.SetWindowSize(image.Pt(400, 400))
	defer elements.SetWindowSize(image.Point{})

	file := New().Item("New", func() {}).Item("Open", func() {})
	edit := New().Item("Undo", func() {}).Item("Redo", func() {})
	bar := NewBar()
	row := elements.Row(styles.Styles{Gap: 10},
		bar.Add(file, elements.Box(styles.Styles{Width: 60, Height: 30})),
		bar.Add(edit, elements.Box(styles.Styles{Width: 60, Height: 30})),
	)

	var ops op.Ops
	var router input.Router
	gtx := layout.Context{Ops: &ops, Source: router.Source(), Constraints: layout.Exact(image.Pt(400, 400))}
	frame := func() {
		ops.Reset()
		row.Layout(gtx)
		file.Layout(gtx)
		edit.Layout(gtx)
		elements.DrainPortals(gtx)
		router.Frame(&ops)
	}
	click := func(x, y float32) {
		p := f32.Pt(x, y)
		router.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: p, Buttons: pointer.ButtonPrimary},
			pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: p})
		frame()
		frame()
	}
	move := func(x, y float32) {
		router.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: f32.Pt(x, y)})
		frame()
		frame()
	}
	state := func() (bool, bool) { return file.Opened(), edit.Opened() }

	frame()
	frame()
	move(300, 300)
	if f, e := state(); f || e {
		t.Fatal("hovering opened a menu of a bar that opens on click")
	}
	click(10, 10)
	if f, e := state(); !f || e {
		t.Fatalf("click on File: file %v, edit %v", f, e)
	}
	// With one open, moving onto the other trigger switches.
	move(100, 10)
	if f, e := state(); f || !e {
		t.Fatalf("pointer onto Edit: file %v, edit %v; want Edit open instead", f, e)
	}
	if edit.pos.X < 70 {
		t.Errorf("Edit's menu at %v, want under its trigger", edit.pos)
	}
	// A click on another trigger switches too, rather than only closing.
	click(10, 10)
	if f, e := state(); !f || e {
		t.Fatalf("click on File with Edit open: file %v, edit %v", f, e)
	}
	// Right goes to the next menu, wrapping around.
	for _, want := range []bool{false, true} {
		router.Queue(key.Event{Name: key.NameRightArrow, State: key.Press}, key.Event{Name: key.NameRightArrow, State: key.Release})
		frame()
		frame()
		if f, _ := state(); f != want {
			t.Fatalf("after Right: file open %v, want %v", f, want)
		}
	}
	// A click on the open menu's trigger closes it; outside closes too.
	click(10, 10)
	if f, e := state(); f || e {
		t.Fatalf("click on the open menu's trigger: file %v, edit %v; want both closed", f, e)
	}

	// Open on hover: resting on a trigger opens its menu.
	bar.OpenOnHover(true)
	move(300, 300)
	move(100, 10)
	if f, e := state(); f || !e {
		t.Fatalf("hover-open bar, pointer on Edit: file %v, edit %v", f, e)
	}
	move(10, 10)
	if f, e := state(); !f || e {
		t.Fatalf("hover-open bar, pointer onto File: file %v, edit %v", f, e)
	}
	// Clicking the open menu's trigger closes it, and resting there
	// doesn't reopen it.
	click(10, 10)
	if f, e := state(); f || e {
		t.Fatalf("hover-open bar, click on File's trigger: file %v, edit %v; want closed", f, e)
	}
	click(300, 300)
	if f, e := state(); f || e {
		t.Fatalf("click outside: file %v, edit %v; want both closed", f, e)
	}
}
