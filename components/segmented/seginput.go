// Package segmented provides a segmented control: a row of mutually
// exclusive options in one rounded track, as iOS and Android use for
// switching views or modes.

package segmented

import (
	"gioui.org/layout"
	"github.com/xmhrnx/giode/components/kit"
)

// update handles clicks, reporting whether one chose an option.
func (c *Control) update(gtx layout.Context) bool {
	chose := false
	for i := range c.clicks {
		if c.clicks[i].Clicked(gtx) {
			c.Choose(i)
			kit.RequestFrame(gtx)
			chose = true
		}
	}
	return chose
}

func boolString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
