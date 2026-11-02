package dialog

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

func TestDialogTakesNoSpace(t *testing.T) {
	fonts.UseGofont()
	open := true
	d := New(&open, func() elements.Element { return elements.Text("Hello") })
	win := image.Pt(400, 300)
	var r input.Router
	var page layout.Dimensions
	var nodes []elements.SemanticNode
	frame := func() {
		elements.SetWindowSize(win)
		elements.BeginFrame()
		elements.BeginSemantics()
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Source: r.Source(), Constraints: layout.Constraints{Max: win}}
		// The dialog sits in a narrow column, below some content.
		page = elements.Box(styles.Styles{Width: 120},
			elements.Stack(styles.Styles{},
				elements.Box(styles.Styles{Width: 100, Height: 50}),
				elements.Raw(d.Layout),
			)).Layout(gtx)
		elements.DrainPortals(gtx)
		elements.DispatchBack(gtx)
		elements.RunAfterLayout()
		nodes = elements.EndSemantics()
		r.Frame(&ops)
	}
	frame()
	frame()
	if page.Size.Y != 50 {
		t.Errorf("page height %d with the dialog open, want 50", page.Size.Y)
	}
	var text elements.SemanticNode
	for _, n := range nodes {
		if n.Label == "Hello" {
			text = n
		}
	}
	if c := text.Bounds.Min.Add(text.Bounds.Max).Div(2); c.X < 150 || c.X > 250 || c.Y < 100 || c.Y > 200 {
		t.Errorf("dialog content at %v, want centered in the %v window", text.Bounds, win)
	}
	p := f32.Pt(5, 290)
	r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: p, Buttons: pointer.ButtonPrimary},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: p})
	frame()
	if open {
		t.Error("a click on the scrim left the dialog open")
	}
}

func TestDialogOutsideGap(t *testing.T) {
	open := true
	d := New(&open, func() elements.Element { return elements.Text("Hi") })
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: layout.Constraints{Max: image.Pt(400, 300)}}
	elements.BeginFrame()
	dims := elements.Stack(styles.Styles{Gap: 20},
		elements.Box(styles.Styles{Width: 100, Height: 50}),
		d,
	).Layout(gtx)
	elements.DrainPortals(gtx)
	if dims.Size.Y != 50 {
		t.Errorf("height %d, want 50: the gap went around the dialog", dims.Size.Y)
	}
}

func TestZeroPadding(t *testing.T) {
	fonts.UseGofont()
	open := true
	d := New(&open, func() elements.Element { return elements.Box(styles.Styles{Width: 100, Height: 50}) }).Padding(properties.Inset{})
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: layout.Constraints{Max: image.Pt(400, 300)}}
	elements.SetWindowSize(image.Pt(400, 300))
	for range 2 {
		elements.BeginFrame()
		d.Layout(gtx)
		elements.DrainPortals(gtx)
	}
	if d.size != image.Pt(100, 50) {
		t.Errorf("panel %v with Padding(Inset{}), want the content's 100x50", d.size)
	}
	d2 := New(&open, func() elements.Element { return elements.Box(styles.Styles{Width: 100, Height: 50}) }, styles.Styles{Padding: properties.NoInset})
	for range 2 {
		elements.BeginFrame()
		d2.Layout(gtx)
		elements.DrainPortals(gtx)
	}
	if d2.size != image.Pt(100, 50) {
		t.Errorf("panel %v with Padding: NoInset, want the content's 100x50", d2.size)
	}
}
