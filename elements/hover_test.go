package elements

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/styles"
)

// TestBoxOnHover drives a real input router and checks that the hover
// state follows pointer moves into and out of the element.
func TestBoxOnHover(t *testing.T) {
	var ops op.Ops
	var router input.Router
	gtx := layout.Context{
		Ops:         &ops,
		Source:      router.Source(),
		Constraints: layout.Exact(image.Pt(100, 100)),
	}

	el := Box(styles.Styles{Width: -1, Height: -1}).OnHover(styles.Styles{
		Opacity: 0.5, // any non-zero hover marker
	})

	moveTo := func(p f32.Point) {
		router.Queue(pointer.Event{
			Kind:     pointer.Move,
			Source:   pointer.Mouse,
			Position: p,
		})
	}
	frame := func() {
		ops.Reset()
		el.Layout(gtx)
		router.Frame(&ops)
	}

	// Frame 1: register the hover region.
	frame()

	// Move the pointer inside the box.
	moveTo(f32.Pt(10, 10))
	frame() // processes the move: enter
	frame() // settle
	if !el.Hovered() {
		t.Fatalf("pointer inside: Hovered() = false, want true")
	}

	// Move the pointer outside the box.
	moveTo(f32.Pt(200, 200))
	frame()
	frame()
	if el.Hovered() {
		t.Fatalf("pointer outside: Hovered() = true, want false")
	}
}

// TestTextOnHover mirrors TestBoxOnHover for TextElement.
func TestTextOnHover(t *testing.T) {
	fonts.UseGofont()

	var ops op.Ops
	var router input.Router
	gtx := layout.Context{
		Ops:         &ops,
		Source:      router.Source(),
		Constraints: layout.Exact(image.Pt(100, 100)),
	}

	el := Text("hover me", styles.Styles{}).OnHover(styles.Styles{
		Opacity: 0.5, // any non-zero hover marker
	})

	moveTo := func(p f32.Point) {
		router.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: p})
	}
	frame := func() {
		ops.Reset()
		el.Layout(gtx)
		router.Frame(&ops)
	}

	frame() // register the hover region

	moveTo(f32.Pt(5, 5))
	frame()
	frame()
	if !el.Hovered() {
		t.Fatalf("pointer inside: Hovered() = false, want true")
	}

	moveTo(f32.Pt(200, 200))
	frame()
	frame()
	if el.Hovered() {
		t.Fatalf("pointer outside: Hovered() = true, want false")
	}
}

// TestDividerOnHover mirrors TestBoxOnHover for DividerElement.
func TestDividerOnHover(t *testing.T) {
	var ops op.Ops
	var router input.Router
	gtx := layout.Context{
		Ops:         &ops,
		Source:      router.Source(),
		Constraints: layout.Exact(image.Pt(100, 20)),
	}

	el := Divider(styles.Styles{}).OnHover(styles.Styles{
		Opacity: 0.5, // any non-zero hover marker
	})

	moveTo := func(p f32.Point) {
		router.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: p})
	}
	frame := func() {
		ops.Reset()
		el.Layout(gtx)
		router.Frame(&ops)
	}

	frame()

	// The divider defaults to 1px tall, so the pointer must land within
	// that thin strip.
	moveTo(f32.Pt(5, 0))
	frame()
	frame()
	if !el.Hovered() {
		t.Fatalf("pointer inside: Hovered() = false, want true")
	}

	moveTo(f32.Pt(500, 500))
	frame()
	frame()
	if el.Hovered() {
		t.Fatalf("pointer outside: Hovered() = true, want false")
	}
}
