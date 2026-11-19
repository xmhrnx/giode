package platform

import (
	"image"
	"runtime"
	"testing"

	"gioui.org/layout"

	"github.com/xmhrnx/giode/elements/internal/base"
)

type sized struct{ w int }

func (s sized) Layout(gtx layout.Context) layout.Dimensions {
	return layout.Dimensions{Size: image.Pt(s.w, 10)}
}

func TestCurrentMatchesGOOS(t *testing.T) {
	want := map[string]Platform{"windows": Windows, "darwin": MacOS, "linux": Linux, "android": Android, "ios": IOS, "js": Web}[runtime.GOOS]
	if want != 0 && Current != want {
		t.Errorf("Current = %v on %s", Current, runtime.GOOS)
	}
}

func TestOnlyOn(t *testing.T) {
	calls := 0
	build := func() base.Element { calls++; return sized{5} }
	gtx := layout.Context{}
	if d := OnlyOn(Any&^Current, build).Layout(gtx); d.Size != (image.Point{}) || calls != 0 {
		t.Errorf("other platforms: size %v, %d builds", d.Size, calls)
	}
	if d := OnlyOn(Current, build).Layout(gtx); d.Size.X != 5 || calls != 1 {
		t.Errorf("this platform: size %v, %d builds", d.Size, calls)
	}
}

func TestByPlatform(t *testing.T) {
	gtx := layout.Context{}
	el := func(w int) func() base.Element { return func() base.Element { return sized{w} } }
	v := Views{Windows: el(1), MacOS: el(1), Linux: el(1), Android: el(1), IOS: el(1), Web: el(1), Desktop: el(2), Mobile: el(2), Default: el(3)}
	if w := ByPlatform(v).Layout(gtx).Size.X; w != 1 {
		t.Errorf("specific: width %d", w)
	}
	v = Views{Desktop: el(2), Mobile: el(2), Default: el(3)}
	if w := ByPlatform(v).Layout(gtx).Size.X; Current != Web && w != 2 {
		t.Errorf("group: width %d", w)
	}
	if w := ByPlatform(Views{Default: el(3)}).Layout(gtx).Size.X; w != 3 {
		t.Errorf("default: width %d", w)
	}
	if w := ByPlatform(Views{}).Layout(gtx).Size.X; w != 0 {
		t.Errorf("none: width %d", w)
	}
}
