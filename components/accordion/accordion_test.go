package accordion

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

func TestAccordion_ClickTogglesSection(t *testing.T) {
	built := 0
	a := New().Item("Section A", func() elements.Element {
		built++
		return elements.Text("content A")
	})

	var ops op.Ops
	var router input.Router
	gtx := layout.Context{Ops: &ops, Source: router.Source(), Constraints: layout.Exact(image.Pt(300, 300))}

	frame := func() {
		ops.Reset()
		a.Layout(gtx)
		router.Frame(&ops)
	}
	press := func(p f32.Point) {
		router.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: p, Buttons: pointer.ButtonPrimary})
	}
	release := func(p f32.Point) {
		router.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: p, Buttons: 0})
	}

	frame() // register the header's clickable area
	if a.Expanded(0) {
		t.Fatal("new accordion: Expanded(0) = true, want false")
	}
	if built != 0 {
		t.Fatalf("content built = %d times before being expanded, want 0 (built lazily)", built)
	}

	press(f32.Pt(10, 10))
	frame()
	release(f32.Pt(10, 10))
	frame()
	if !a.Expanded(0) {
		t.Fatal("after clicking the header: Expanded(0) = false, want true")
	}
	if built != 1 {
		t.Errorf("content built = %d times after expanding, want 1", built)
	}

	press(f32.Pt(10, 10))
	frame()
	release(f32.Pt(10, 10))
	frame()
	if a.Expanded(0) {
		t.Fatal("after clicking the header again: Expanded(0) = true, want false")
	}
}

func TestAccordion_SingleOpenClosesOthers(t *testing.T) {
	a := New().SingleOpen(true).
		Item("A", func() elements.Element { return elements.Text("a") }).
		Item("B", func() elements.Element { return elements.Text("b") })

	a.SetExpanded(0, true)
	if !a.Expanded(0) {
		t.Fatal("SetExpanded(0, true): Expanded(0) = false, want true")
	}

	a.SetExpanded(1, true)
	if a.Expanded(0) {
		t.Error("after opening section 1 with SingleOpen: section 0 still expanded, want closed")
	}
	if !a.Expanded(1) {
		t.Error("after opening section 1 with SingleOpen: Expanded(1) = false, want true")
	}
}

func TestAccordion_IndependentByDefault(t *testing.T) {
	a := New().
		Item("A", func() elements.Element { return elements.Text("a") }).
		Item("B", func() elements.Element { return elements.Text("b") })

	a.SetExpanded(0, true)
	a.SetExpanded(1, true)
	if !a.Expanded(0) || !a.Expanded(1) {
		t.Error("without SingleOpen, both sections should be able to stay expanded at once")
	}
}

func TestAccordion_ContentLaysOutWhileExpanded(t *testing.T) {
	a := New().Item("A", func() elements.Element {
		return elements.Box(styles.Styles{Width: 50, Height: 20, Background: "black"})
	})
	a.SetExpanded(0, true)

	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: layout.Exact(image.Pt(300, 300))}
	dims := a.Layout(gtx)
	// The header alone is much shorter than header+content; a rough
	// floor confirms the content actually got laid out, not skipped.
	if dims.Size.Y < 40 {
		t.Errorf("Layout height = %d with content expanded, want header+content (>= 40)", dims.Size.Y)
	}
}
