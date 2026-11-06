package text_test

import (
	"image"
	"strings"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// MaxWidth caps a Text or RichText, ellipsis or wrapping, as it caps a
// Box - also in a parent that stretches it; MinWidth widens a short one.
func TestTextMinMaxWidth(t *testing.T) {
	fonts.UseGofont()
	long := strings.Repeat("A long line of text ", 10)
	ellipsis := styles.Styles{MaxWidth: 120, TextOverflow: properties.TextOverflowEllipsis, FontSize: 14}
	cases := map[string]struct {
		el       func() elements.Element
		min, max int
	}{
		"text ellipsis": {func() elements.Element { return elements.Text(long, ellipsis) }, 1, 240},
		"text wraps":    {func() elements.Element { return elements.Text(long, styles.Styles{MaxWidth: 120}) }, 1, 240},
		"rich text": {func() elements.Element {
			return elements.RichText(styles.Styles{MaxWidth: 120}, elements.SpanText(long))
		}, 1, 240},
		// At least 150dp; stretched, as wide as the parent.
		"min width": {func() elements.Element { return elements.Text("Hi", styles.Styles{MinWidth: 150}) }, 300, 1000},
	}
	for name, c := range cases {
		for _, stretch := range []bool{false, true} {
			var got int
			el := c.el()
			probe := elements.Raw(func(gtx layout.Context) layout.Dimensions {
				d := el.Layout(gtx)
				got = d.Size.X
				return d
			})
			st := styles.Styles{Width: -1}
			if stretch {
				st.Align = properties.AlignStretch
			}
			var ops op.Ops
			gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 2, PxPerSp: 2}, Constraints: layout.Constraints{Max: image.Pt(1000, 1000)}}
			elements.Stack(st, probe).Layout(gtx)
			if got < c.min || got > c.max {
				t.Errorf("%s (stretched %v): width %dpx, want %d to %d", name, stretch, got, c.min, c.max)
			}
		}
	}
}
