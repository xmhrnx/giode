package text

import (
	"image"
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"gioui.org/f32"
	"gioui.org/io/pointer"
	"gioui.org/layout"

	"github.com/xmhrnx/giode/internal/textrender"
)

// selArea is a selection area's persistent state.
type selArea struct {
	// parts are being collected this frame; prev are last frame's,
	// whose positions are exact by the time events arrive.
	parts, prev []*selPart
	// order holds, for each part this frame, the order it registered
	// in, as containers that lay children out of screen order
	// rearrange parts (see base.ReorderTracked); regPlace maps last
	// frame's registration order to the part's place, to paint a
	// part's selection as it registers.
	order, regPlace []int
	origin          image.Point

	anchor, focus selPos
	has           bool
	// version counts selection changes; seen is the version the area
	// was last laid out with.
	version, seen uint64
	collecting    bool

	tag       int
	dragging  bool
	clicks    int
	lastClick time.Time
	lastPos   f32.Point

	// Touch: a press held for longPress starts selecting.
	pressing bool
	touchSel bool
	pressAt  time.Time
	pressPos f32.Point
	pressID  pointer.ID

	showCopy bool
	copy     *copyAction
	pass     func(gtx layout.Context) bool
}

func newSelArea() *selArea {
	return &selArea{copy: newCopyAction()}
}

// rangeOf returns the runes of part idx inside the selection.
func (s *selArea) rangeOf(idx int, text string) (lo, hi int) {
	if !s.has {
		return 0, 0
	}
	a, b := s.ordered()
	if idx < a.part || idx > b.part {
		return 0, 0
	}
	lo, hi = 0, math.MaxInt
	if idx == a.part {
		lo = a.rune
	}
	if idx == b.part {
		hi = b.rune
	}
	if hi == math.MaxInt {
		hi = utf8.RuneCountInString(text)
	}
	return lo, hi
}

func (s *selArea) ordered() (selPos, selPos) {
	if s.focus.less(s.anchor) {
		return s.focus, s.anchor
	}
	return s.anchor, s.focus
}

// empty reports whether nothing is selected.
func (s *selArea) empty() bool {
	return !s.has || s.anchor == s.focus
}

// selectedText returns the selected text, the parts joined by newlines.
func (s *selArea) selectedText() string {
	if s.empty() {
		return ""
	}
	a, b := s.ordered()
	var out strings.Builder
	for i := a.part; i <= b.part && i < len(s.prev); i++ {
		lo, hi := s.rangeOf(i, s.prev[i].text)
		if i > a.part {
			out.WriteByte('\n')
		}
		out.WriteString(runeSlice(s.prev[i].text, lo, hi))
	}
	return out.String()
}

// hit returns the caret position nearest p, in window coordinates: in
// the part under p, or the nearest part.
func (s *selArea) hit(p image.Point) (selPos, bool) {
	best, bestD := -1, 0
	for i, part := range s.prev {
		r := image.Rectangle{Min: part.origin, Max: part.origin.Add(part.size)}
		dx := max(r.Min.X-p.X, 0, p.X-r.Max.X)
		dy := max(r.Min.Y-p.Y, 0, p.Y-r.Max.Y)
		// Lines matter more than columns: a point beside a paragraph
		// belongs to it, not to the one diagonally nearer.
		d := dy*8 + dx
		if best < 0 || d < bestD {
			best, bestD = i, d
		}
	}
	if best < 0 {
		return selPos{}, false
	}
	part := s.prev[best]
	return selPos{part: best, rune: textrender.HitRune(part.stopLines(), p.Sub(part.origin))}, true
}

// word returns the word around pos, or the rune at it.
func (s *selArea) word(pos selPos) (selPos, selPos) {
	rs := []rune(s.prev[pos.part].text)
	isWord := func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r) || r == '_' }
	i := min(pos.rune, len(rs))
	if i == len(rs) || !isWord(rs[i]) {
		if i > 0 && isWord(rs[i-1]) {
			i--
		} else {
			return pos, selPos{pos.part, min(i+1, len(rs))}
		}
	}
	lo, hi := i, i
	for lo > 0 && isWord(rs[lo-1]) {
		lo--
	}
	for hi < len(rs) && isWord(rs[hi]) {
		hi++
	}
	return selPos{pos.part, lo}, selPos{pos.part, hi}
}

func (s *selArea) set(anchor, focus selPos) {
	if !s.has || s.anchor != anchor || s.focus != focus {
		s.anchor, s.focus, s.has = anchor, focus, true
		s.version++
	}
}

func (s *selArea) clear() {
	if s.has {
		s.has = false
		s.version++
	}
	s.showCopy = false
}

func (s *selArea) abs(p f32.Point) image.Point {
	return s.origin.Add(p.Round())
}

// bounds returns the selection's bounds in window coordinates.
func (s *selArea) bounds() image.Rectangle {
	a, b := s.ordered()
	var r image.Rectangle
	for _, i := range []int{a.part, b.part} {
		if i >= len(s.prev) {
			continue
		}
		p := s.prev[i]
		lo, hi := s.rangeOf(i, p.text)
		for _, rr := range textrender.RangeRects(p.stopLines(), lo, hi) {
			r = r.Union(rr.Add(p.origin))
		}
	}
	return r
}

// placeOf is where the part registering reg-th this frame ends up, as
// it did last frame.
func (s *selArea) placeOf(reg int) int {
	if reg < len(s.regPlace) {
		return s.regPlace[reg]
	}
	return reg
}

// reorder puts the parts in spans in the order the spans are given,
// where the spans lie now.
func (s *selArea) reorder(spans [][2]int) {
	lo := spans[0][0]
	for _, sp := range spans {
		lo = min(lo, sp[0])
	}
	parts := make([]*selPart, 0, len(s.parts))
	order := make([]int, 0, len(s.order))
	for _, sp := range spans {
		parts = append(parts, s.parts[sp[0]:sp[1]]...)
		order = append(order, s.order[sp[0]:sp[1]]...)
	}
	copy(s.parts[lo:], parts)
	copy(s.order[lo:], order)
}

// endFrame records where each registration ended up, for the next
// frame's painting.
func (s *selArea) endFrame() {
	s.regPlace = s.regPlace[:0]
	for range s.order {
		s.regPlace = append(s.regPlace, 0)
	}
	for pos, reg := range s.order {
		if reg < len(s.regPlace) {
			s.regPlace[reg] = pos
		}
	}
	s.order = s.order[:0]
}
