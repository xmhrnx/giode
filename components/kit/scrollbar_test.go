package kit

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/styles/properties"
)

func TestScrollbar_Hidden(t *testing.T) {
	defer func(f func() (int, string)) { Screen = f }(Screen)
	width, bp := 0, ""
	Screen = func() (int, string) { return width, bp }
	gtx := layout.Context{Ops: new(op.Ops), Metric: unit.Metric{PxPerDp: 2, PxPerSp: 2}}

	shown := func(sb properties.ScrollBar) bool {
		var s Scrollbar
		off := float32(0)
		s.Layout(gtx, image.Pt(200, 100), 1000, &off, sb)
		return s.shown
	}
	cases := []struct {
		name  string
		width int
		bp    string
		sb    properties.ScrollBar
		want  bool
	}{
		{"default", 1600, "", properties.ScrollBar{}, true},
		{"hidden", 1600, "", properties.ScrollBar{Hidden: true}, false},
		// 600dp is 1200px at 2px per dp.
		{"phone width", 1000, "", properties.ScrollBar{HiddenMaxWidth: 600}, false},
		{"desktop width", 1600, "", properties.ScrollBar{HiddenMaxWidth: 600}, true},
		{"compact breakpoint", 1600, "compact", properties.ScrollBar{HiddenOn: "compact"}, false},
		{"one of several", 1600, "medium", properties.ScrollBar{HiddenOn: "compact, medium"}, false},
		{"other breakpoint", 1600, "wide", properties.ScrollBar{HiddenOn: "compact,medium"}, true},
	}
	for _, c := range cases {
		width, bp = c.width, c.bp
		if got := shown(c.sb); got != c.want {
			t.Errorf("%s: shown = %v, want %v", c.name, got, c.want)
		}
	}
}
