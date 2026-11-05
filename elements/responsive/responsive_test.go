package responsive

import (
	"image"
	"testing"

	"gioui.org/unit"

	"github.com/xmhrnx/giode/elements/internal/base"
)

func TestCompactOutsideResponsive(t *testing.T) {
	defer base.SetWindowSize(base.WindowSize())
	base.SetMetric(unit.Metric{PxPerDp: 2})
	defer base.SetMetric(unit.Metric{PxPerDp: 1})
	base.SetWindowSize(image.Pt(1000, 800)) // 500 dp
	if !base.Compact() {
		t.Error("a 500 dp window isn't compact")
	}
	base.SetWindowSize(image.Pt(1400, 800)) // 700 dp
	if base.Compact() {
		t.Error("a 700 dp window is compact")
	}
}
