package chip

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/components/badge"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/styles"
)

func TestChipClickAndRemove(t *testing.T) {
	fonts.UseGofont()
	clicks, removed := 0, 0
	c := New("Go").OnClick(func() { clicks++ }).OnRemove(func() { removed++ })
	var r input.Router
	var dims layout.Dimensions
	frame := func() {
		var ops op.Ops
		dims = c.Layout(layout.Context{Ops: &ops, Source: r.Source(), Constraints: layout.Constraints{Max: image.Pt(300, 100)}})
		r.Frame(&ops)
	}
	tap := func(x float32) {
		p := f32.Pt(x, float32(dims.Size.Y/2))
		r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: p, Buttons: pointer.ButtonPrimary},
			pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: p})
		frame()
		frame()
	}
	frame()
	tap(8)
	tap(float32(dims.Size.X - 16)) // the remove button
	if clicks != 1 || removed != 1 {
		t.Errorf("clicks=%d removed=%d, want 1 and 1", clicks, removed)
	}
}

func TestBadgeOnKeepsTargetSize(t *testing.T) {
	fonts.UseGofont()
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: layout.Constraints{Max: image.Pt(300, 300)}}
	d := badge.New("3").On(elements.Box(styles.Styles{Width: 40, Height: 40})).Layout(gtx)
	if d.Size != image.Pt(40, 40) {
		t.Errorf("badged element %v, want the target's 40x40", d.Size)
	}
}

func TestSelectedKeepsItsColors(t *testing.T) {
	c := New("Go", styles.Styles{Background: "#222222", Color: "#eeeeee", BorderColor: "#444444"})
	if st := c.styles(); st.Background != "#222222" || st.Color != "#eeeeee" {
		t.Errorf("unselected %q on %q, want the given colors", st.Color, st.Background)
	}
	st := c.Selected(true).styles()
	if st.Background != "theme.primary" || st.Color != "theme.onprimary" || st.BorderColor != "theme.primary" {
		t.Errorf("selected %q on %q, border %q: styles overrode the selected state", st.Color, st.Background, st.BorderColor)
	}
	if st := c.SelectedStyles(styles.Styles{Background: "#0000ff"}).styles(); st.Background != "#0000ff" || st.Color != "theme.onprimary" {
		t.Errorf("SelectedStyles gave %q on %q", st.Color, st.Background)
	}
}
