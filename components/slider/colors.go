package slider

import (
	"gioui.org/layout"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/styles/properties"
)

var (
	defaultTrack = "#334155"
	defaultFill  = "#3b82f6"
	defaultThumb = "#f8fafc"
)

const (
	height = 24
	trackH = 4
	thumbD = 14
)

// metrics is the slider's pixel geometry for one frame: the dp
// constants above resolved through the frame's content scale, so the
// slider keeps its physical size on HiDPI displays.
type metrics struct {
	height  int
	trackH  int
	thumbD  int
	padding int
}

// metricsFor scales the dp constants with gtx's metric.
func metricsFor(gtx layout.Context) metrics {
	m := metrics{
		height: gtx.Dp(unit.Dp(height)),
		trackH: gtx.Dp(unit.Dp(trackH)),
		thumbD: gtx.Dp(unit.Dp(thumbD)),
	}
	m.padding = m.thumbD / 2
	return m
}

func resolve(c, def properties.Color) properties.Color {
	if c == "" {
		return def
	}
	return c
}
