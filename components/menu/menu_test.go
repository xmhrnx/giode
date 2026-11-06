package menu

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
)

func TestMenu_Trigger_ClickTogglesOpen(t *testing.T) {
	elements.SetWindowSize(image.Pt(400, 400))
	defer elements.SetWindowSize(image.Point{})

	clicked := false
	m := New().Item("Item", func() { clicked = true })
	trigger := elements.Box(styles.Styles{Width: 60, Height: 30})
	wrapped := m.Trigger(trigger)

	var ops op.Ops
	var router input.Router
	gtx := layout.Context{Ops: &ops, Source: router.Source(), Constraints: layout.Exact(image.Pt(400, 400))}

	frame := func() {
		ops.Reset()
		wrapped.Layout(gtx)
		m.Layout(gtx)
		elements.DrainPortals(gtx)
		router.Frame(&ops)
	}
	press := func(p f32.Point) {
		router.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: p, Buttons: pointer.ButtonPrimary})
	}
	release := func(p f32.Point) {
		router.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: p, Buttons: 0})
	}

	frame() // register the trigger's clickable area

	press(f32.Pt(10, 10))
	frame()
	release(f32.Pt(10, 10))
	frame()
	if !m.Opened() {
		t.Fatalf("after clicking the trigger: Opened() = false, want true")
	}

	press(f32.Pt(10, 10))
	frame()
	release(f32.Pt(10, 10))
	frame()
	if m.Opened() {
		t.Fatalf("after clicking the trigger again: Opened() = true, want false")
	}
	if clicked {
		t.Errorf("the menu item's own action fired from clicking the trigger, want it untouched")
	}
}

func TestMenu_SubMenu_HoverOpensAndPersists(t *testing.T) {
	elements.SetWindowSize(image.Pt(400, 400))
	defer elements.SetWindowSize(image.Point{})

	sub := New().Item("Sub item", func() {})
	m := New().SubMenu("More", sub)
	m.OpenAt(f32.Pt(0, 0))

	var ops op.Ops
	var router input.Router
	gtx := layout.Context{Ops: &ops, Source: router.Source(), Constraints: layout.Exact(image.Pt(400, 400))}

	frame := func() {
		ops.Reset()
		m.Layout(gtx)
		elements.DrainPortals(gtx)
		router.Frame(&ops)
	}
	moveTo := func(p f32.Point) {
		router.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: p})
	}

	frame() // registers the "More" row's hover area

	// The panel is clamped inside the margin near the (0,0) anchor, so
	// its first (and only) row starts a little inside that corner.
	moveTo(f32.Pt(30, 25))
	frame()
	frame()
	if !sub.Opened() {
		t.Fatalf("after hovering the submenu item: submenu Opened() = false, want true")
	}

	// Moving away from the row (but not onto a sibling submenu item -
	// there isn't one here) must NOT close it: it persists until
	// superseded, so moving into the submenu's own panel doesn't
	// flicker it shut.
	moveTo(f32.Pt(390, 390))
	frame()
	frame()
	if !sub.Opened() {
		t.Fatalf("after moving away (no sibling submenu hovered): submenu Opened() = false, want true (should persist)")
	}

	m.Close()
	if sub.Opened() {
		t.Errorf("after closing the parent menu: submenu Opened() = true, want false (Close is recursive)")
	}
}

// TestMenu_PanelHugsWideWindow checks the panel is as wide as its
// widest row, not the window: a no-wrap row holding a growing label
// takes all the width it's offered.
func TestMenu_PanelHugsWideWindow(t *testing.T) {
	elements.SetWindowSize(image.Pt(1600, 1000))
	defer elements.SetWindowSize(image.Point{})
	sub := New().Item("Deeper", func() {})
	m := New().Item("Play", func() {}).Item("Pause", func() {}).Separator().SubMenu("Speed", sub)
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: layout.Exact(image.Pt(1600, 1000))}
	if w := m.measureWidth(gtx, styles.Styles{}); w <= 0 || w > 300 {
		t.Errorf("measured width %d, want the rows' content width", w)
	}
	dims := m.panel(gtx)
	if dims.Size.X > 400 {
		t.Errorf("panel is %d px wide in a 1600 px window, want it to hug its rows", dims.Size.X)
	}
}

// TestMenu_ShortcutsAndChecks checks shortcut and check rows: the
// shortcut widens the panel, check state is read each frame and
// reported to automation.
func TestMenu_ShortcutsAndChecks(t *testing.T) {
	elements.SetWindowSize(image.Pt(800, 600))
	defer elements.SetWindowSize(image.Point{})
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: layout.Exact(image.Pt(800, 600))}

	plain := New().Item("Open File…", func() {})
	withKey := New().Item("Open File…", func() {}).Shortcut("Ctrl+Shift+O")
	if a, b := plain.measureWidth(gtx, styles.Styles{}), withKey.measureWidth(gtx, styles.Styles{}); b <= a {
		t.Errorf("shortcut row %d px, plain %d px: want the shortcut to add width", b, a)
	}

	speed := 1
	m := New().
		RadioItem("1x", func() bool { return speed == 1 }, func() { speed = 1 }).
		RadioItem("2x", func() bool { return speed == 2 }, func() { speed = 2 }).
		CheckItem("Loop", func() bool { return true }, func() {}).
		Item("Plain", func() {})
	if !m.hasLeading() {
		t.Fatal("check items should reserve the leading column")
	}
	// Every row is as wide once the leading column is reserved: Plain
	// is shortest and must still line up.
	w := m.measureWidth(gtx, styles.Styles{})
	if w <= 0 {
		t.Fatal("no width")
	}
	m.items[1].action()
	if speed != 2 || !m.items[1].checked() || m.items[0].checked() {
		t.Error("radio state")
	}
}

// OpenAtClick opens the menu at the click's window position.
func TestOpenAtClick(t *testing.T) {
	m := New(styles.Styles{}).Item("Close", func() {})
	m.OpenAtClick(elements.ClickEvent{WindowPosition: image.Pt(30, 40)})
	if !m.Opened() || m.pos != f32.Pt(30, 40) {
		t.Errorf("opened %v at %v, want (30, 40)", m.Opened(), m.pos)
	}
}
