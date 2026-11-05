package text

import (
	"gioui.org/gesture"

	"github.com/xmhrnx/giode/elements/internal/base"
)

// linkKey identifies a link's click state across frames.
type linkKey struct {
	url, text string
	n         int
}

// linkCount numbers repeated identical links within a frame, so each
// keeps its own click state.
var (
	linkCount      = map[linkKey]int{}
	linkCountFrame uint64
)

func linkClick(url, text string) *gesture.Click {
	if linkCountFrame != base.FrameNumber() {
		clear(linkCount)
		linkCountFrame = base.FrameNumber()
	}
	k := linkKey{url: url, text: text}
	k.n = linkCount[k]
	linkCount[linkKey{url: url, text: text}]++
	return base.Keyed("giode.richtext.link", k, func() *gesture.Click { return new(gesture.Click) })
}
