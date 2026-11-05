package text

import (
	"image"

	"github.com/xmhrnx/giode/internal/textrender"
)

// selPart is a text element inside a selection area, as laid out in
// the last frame.
type selPart struct {
	origin image.Point
	size   image.Point
	text   string
	stops  func() []textrender.StopLine
	lines  []textrender.StopLine
	cached bool
}

// selPos is a caret position: a part and a rune offset in its text.
type selPos struct{ part, rune int }

func (p *selPart) stopLines() []textrender.StopLine {
	if !p.cached {
		p.lines, p.cached = p.stops(), true
	}
	return p.lines
}

func (a selPos) less(b selPos) bool {
	return a.part < b.part || a.part == b.part && a.rune < b.rune
}
