package menu

import (
	"image"

	"gioui.org/f32"
	"gioui.org/gesture"
	"gioui.org/layout"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
)

// Bar is a menu bar: a row of menus (File, Edit, View) that behaves as
// one. A click on a trigger opens its menu; once one is open, moving the
// pointer onto another trigger switches to that menu, as does a click
// on it or Left and Right on the keyboard. With OpenOnHover, resting
// the pointer on a trigger opens its menu with none open too.
//
//	bar := menu.NewBar()
//	titleBar.Leading(
//		bar.Add(fileMenu, fileButton),
//		bar.Add(editMenu, editButton),
//		bar.Add(viewMenu, viewButton),
//	)
//
// Each menu must still be laid out somewhere, as with Trigger.
type Bar struct {
	entries   []*menuTrigger
	hoverOpen bool
}

// NewBar returns an empty menu bar.
func NewBar() *Bar {
	return &Bar{}
}

// Add puts m in the bar and returns its trigger: el, which toggles m
// when clicked and switches to m while another of the bar's menus is
// open. Lay it out where el would go.
func (b *Bar) Add(m *Menu, el elements.Element) elements.Element {
	t := m.Trigger(el).(*menuTrigger)
	t.bar = b
	m.bar = b
	b.entries = append(b.entries, t)
	return t
}

// OpenOnHover makes resting the pointer on a trigger open its menu,
// without a click. The menu stays open until something is chosen, a
// click lands outside it, Escape, or the pointer moves to another
// trigger. It returns b for chaining.
func (b *Bar) OpenOnHover(on bool) *Bar {
	b.hoverOpen = on
	return b
}

// Open returns the bar's open menu, or nil.
func (b *Bar) Open() *Menu {
	for _, t := range b.entries {
		if t.menu.open {
			return t.menu
		}
	}
	return nil
}

// at returns the trigger at p, in window pixels, as laid out in the last
// frame, or nil.
func (b *Bar) at(p image.Point) *menuTrigger {
	for _, t := range b.entries {
		r := image.Rectangle{Min: t.origin, Max: t.origin.Add(image.Pt(t.width, t.height))}
		if p.In(r) {
			return t
		}
	}
	return nil
}

// switchTo closes the open menu and opens t's under its trigger.
func (b *Bar) switchTo(t *menuTrigger, keyboard bool) {
	if t.menu.open {
		return
	}
	for _, o := range b.entries {
		if o.menu.open {
			o.menu.Close()
		}
	}
	t.open()
	if keyboard {
		t.menu.moveActive(1)
	}
}

// step switches from m to the menu dir (±1) places along, wrapping.
func (b *Bar) step(m *Menu, dir int) bool {
	for i, t := range b.entries {
		if t.menu == m {
			n := len(b.entries)
			b.switchTo(b.entries[((i+dir)%n+n)%n], true)
			return true
		}
	}
	return false
}

// open opens t's menu below the trigger, where it was laid out last.
func (t *menuTrigger) open() {
	t.menu.open = true
	t.placeMenu()
}

// placeMenu puts the menu just below the trigger: its start edge under
// the trigger's.
func (t *menuTrigger) placeMenu() {
	p := t.origin.Add(image.Pt(0, t.height+4))
	if t.rtl {
		// The menu's right edge under the trigger's.
		p.X += t.width
	}
	t.menu.pos = f32.Pt(float32(p.X), float32(p.Y))
}

// hoverOpens opens the menu when the pointer rests on the trigger of a
// bar that opens on hover.
func (t *menuTrigger) hoverOpens(gtx layout.Context) {
	if t.bar == nil || !t.bar.hoverOpen {
		return
	}
	if t.hover == nil {
		t.hover = new(gesture.Hover)
	}
	// When the pointer arrives: a click that closed the menu while it
	// rests there doesn't reopen it.
	hovered := t.hover.Update(gtx.Source)
	if hovered && !t.hovered && t.bar.Open() == nil {
		t.open()
		kit.RequestFrame(gtx)
	}
	t.hovered = hovered
}
