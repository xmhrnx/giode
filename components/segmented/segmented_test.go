package segmented

import (
	"image"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/styles"
)

func TestSegmented(t *testing.T) {
	fonts.UseGofont()
	var picked []int
	c := New("Chat", "Research", "Code").Styles(styles.Styles{Width: -1}).OnChange(func(i int) { picked = append(picked, i) })
	var r input.Router
	now := time.Unix(0, 0)
	var dims layout.Dimensions
	frame := func() {
		now = now.Add(50 * time.Millisecond)
		var ops op.Ops
		dims = c.Layout(layout.Context{Ops: &ops, Now: now, Source: r.Source(), Constraints: layout.Constraints{Max: image.Pt(300, 100)}})
		r.Frame(&ops)
	}
	frame()
	if dims.Size.X != 300 {
		t.Fatalf("width %d, want the full 300", dims.Size.X)
	}
	p := f32.Pt(250, float32(dims.Size.Y/2)) // the third segment
	r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Touch, Position: p, Buttons: pointer.ButtonPrimary},
		pointer.Event{Kind: pointer.Release, Source: pointer.Touch, Position: p})
	for i := 0; i < 8; i++ {
		frame()
	}
	if c.Selected() != 2 || len(picked) != 1 || picked[0] != 2 {
		t.Errorf("selected %d, picked %v; want 2", c.Selected(), picked)
	}
	if c.pos != 2 {
		t.Errorf("highlight at %v, want 2", c.pos)
	}
}

func TestRTLOrder(t *testing.T) {
	fonts.UseGofont()
	c := New("One", "Two", "Three")
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Locale: system.Locale{Direction: system.RTL}, Constraints: layout.Constraints{Max: image.Pt(400, 100)}}
	elements.BeginSemantics()
	c.Layout(gtx)
	b := map[string]image.Rectangle{}
	for _, n := range elements.EndSemantics() {
		if n.Kind == elements.SemanticTab {
			b[n.Label] = n.Bounds
		}
	}
	if !(b["One"].Min.X > b["Two"].Min.X && b["Two"].Min.X > b["Three"].Min.X) {
		t.Errorf("right to left, One should be rightmost: %v", b)
	}
	if b["One"].Dx() >= 300 {
		t.Errorf("a segment is %d px wide: the options should share the row", b["One"].Dx())
	}
}
