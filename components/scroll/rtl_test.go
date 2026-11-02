package scroll

import (
	"image"
	"strconv"
	"testing"

	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/styles"
)

func TestHorizontalScrollStartsAtTheStart(t *testing.T) {
	fonts.UseGofont()
	for _, dir := range []system.TextDirection{system.LTR, system.RTL} {
		kids := make([]elements.Element, 10)
		for i := range kids {
			kids[i] = elements.Box(styles.Styles{Width: 100, Height: 20}, elements.Text(strconv.Itoa(i)))
		}
		s := New(func() elements.Element { return elements.Row(styles.Styles{}, kids...) }).Axis(Horizontal)
		frame := func() map[string]image.Rectangle {
			var ops op.Ops
			gtx := layout.Context{Ops: &ops, Locale: system.Locale{Direction: dir}, Constraints: layout.Exact(image.Pt(300, 40))}
			elements.BeginSemantics()
			s.Layout(gtx)
			b := map[string]image.Rectangle{}
			for _, n := range elements.EndSemantics() {
				b[n.Label] = n.Bounds
			}
			return b
		}
		frame()
		b := frame()
		first := b["0"]
		if first.Min.X < 0 || first.Max.X > 300 {
			t.Errorf("%v: the first item is at %v, outside the 300px view", dir, first)
		}
		// Scrolling toward the end reveals the last item.
		s.ScrollToX(700)
		b = frame()
		if last := b["9"]; last.Min.X < 0 || last.Max.X > 300 {
			t.Errorf("%v: scrolled to the end, the last item is at %v", dir, last)
		}
	}
}
