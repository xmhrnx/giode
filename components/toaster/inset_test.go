package toaster

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/styles/properties"
)

// Each side of the inset stays on its side, whichever is larger.
func TestInsetSides(t *testing.T) {
	gtx := layout.Context{Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
	tt := New().Inset(properties.Inset{Top: 56, Left: 30})
	if got, want := tt.insetPx(gtx), (image.Rectangle{Min: image.Pt(30, 56)}); got != want {
		t.Errorf("inset %v, want %v", got, want)
	}
	tt.Inset(properties.Inset{Bottom: 64, Right: 12})
	if got, want := tt.insetPx(gtx), (image.Rectangle{Max: image.Pt(12, 64)}); got != want {
		t.Errorf("inset %v, want %v", got, want)
	}
}
