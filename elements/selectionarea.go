package elements

import (
	"image"
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/internal/textrender"
	"github.com/xmhrnx/giode/styles/properties"
)

// SelectionArea makes the Text and RichText elements inside child
// selectable as one, like a web page: with a mouse, drag across them
// to select, double-click for a word, triple-click for a whole element,
// Shift+click to extend, and press Ctrl+C (Cmd+C) to copy; Ctrl+A
// (Cmd+A) selects everything and Escape clears. A right-click on a
// selection offers Copy. On a touch screen, a long-press selects a
// word and dragging the held finger extends it; letting go offers
// Copy. Copied text joins the elements with newlines.
//
// key identifies the area across frames. Text inside keeps its own
// styles; the selection is drawn in its color at 30% opacity. Elements
// that are themselves Selectable keep their own selection and aren't
// part of the area's, nor is the text of buttons, menus, inputs and
// other interactive components (see kit.Unselectable). The area works through Memo and LazyList: cached
// subtrees rebuild only when the selection changes.
func SelectionArea(key any, child Element) Element {
	return &selectionAreaEl{key: key, child: child}
}

type selectionAreaEl struct {
	key   any
	child Element
}

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

func (p *selPart) stopLines() []textrender.StopLine {
	if !p.cached {
		p.lines, p.cached = p.stops(), true
	}
	return p.lines
}

// selPos is a caret position: a part and a rune offset in its text.
type selPos struct{ part, rune int }

func (a selPos) less(b selPos) bool {
	return a.part < b.part || a.part == b.part && a.rune < b.rune
}

// selArea is a selection area's persistent state.
type selArea struct {
	// parts are being collected this frame; prev are last frame's,
	// whose positions are exact by the time events arrive.
	parts, prev []*selPart
	origin      image.Point

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

// selAreaStack holds the selection areas being laid out.
var selAreaStack []*selArea

// selectionVersion is the version of the innermost selection area's
// selection, for caches inside it to notice a change.
func selectionVersion() uint64 {
	if n := len(selAreaStack); n > 0 {
		return selAreaStack[n-1].version
	}
	return 0
}

// registerSelectable adds a laid out text element of the given size
// to the innermost selection area, and paints the part of it that's
// selected, in the selection color for col. It reports whether there
// is an area.
func registerSelectable(gtx layout.Context, text string, size image.Point, stops func() []textrender.StopLine, col properties.Color) bool {
	if len(selAreaStack) == 0 || !kit.TextSelectable() {
		return false
	}
	a := selAreaStack[len(selAreaStack)-1]
	p := &selPart{size: size, text: text, stops: stops}
	TrackPosition(&p.origin)
	idx := len(a.parts)
	add := func() {
		if a.collecting {
			a.parts = append(a.parts, p)
		}
	}
	add()
	// A cache replaying this text registers it again.
	retainKeyed(add)
	if lo, hi := a.rangeOf(idx, text); lo < hi {
		c := properties.CalcColor(selectionColor(col))
		for _, r := range textrender.RangeRects(p.stopLines(), lo, hi) {
			cl := clip.Rect(r).Push(gtx.Ops)
			paint.Fill(gtx.Ops, c)
			cl.Pop()
		}
	}
	cl := clip.Rect{Max: size}.Push(gtx.Ops)
	pointer.CursorText.Add(gtx.Ops)
	cl.Pop()
	return true
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

// runeSlice returns the runes [lo, hi) of str.
func runeSlice(str string, lo, hi int) string {
	start, end := len(str), len(str)
	n := 0
	for i := range str {
		if n == lo {
			start = i
		}
		if n == hi {
			end = i
			break
		}
		n++
	}
	if start > end {
		return ""
	}
	return str[start:end]
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

// update processes the area's events and reports whether the selection
// changed since the area was last laid out.
func (s *selArea) update(gtx layout.Context) bool {
	for {
		ev, ok := gtx.Event(
			pointer.Filter{Target: &s.tag, Kinds: pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel},
			key.FocusFilter{Target: &s.tag},
			key.Filter{Focus: &s.tag, Name: "C", Required: key.ModShortcut},
			key.Filter{Focus: &s.tag, Name: "A", Required: key.ModShortcut},
			key.Filter{Focus: &s.tag, Name: key.NameEscape},
		)
		if !ok {
			break
		}
		switch e := ev.(type) {
		case pointer.Event:
			s.pointer(gtx, e)
		case key.Event:
			if e.State != key.Press {
				break
			}
			switch e.Name {
			case "C":
				if t := s.selectedText(); t != "" {
					kit.WriteClipboard(gtx, t)
				}
			case "A":
				if n := len(s.prev); n > 0 {
					s.set(selPos{}, selPos{n - 1, utf8.RuneCountInString(s.prev[n-1].text)})
				}
			case key.NameEscape:
				s.clear()
			}
		}
	}
	if s.pressing && kit.Scrolling() {
		// A finger resting during a scroll isn't a long-press.
		s.pressing = false
	}
	if s.pressing {
		if gtx.Now.Sub(s.pressAt) >= longPress {
			s.pressing = false
			if pos, ok := s.hit(s.abs(s.pressPos)); ok {
				s.set(s.word(pos))
				s.touchSel = true
				// Keep the finger from scrolling what's underneath.
				gtx.Execute(pointer.GrabCmd{Tag: &s.tag, ID: s.pressID})
			}
		} else {
			kit.RequestFrameAt(gtx, s.pressAt.Add(longPress))
		}
	}
	return s.version != s.seen
}

func (s *selArea) abs(p f32.Point) image.Point {
	return s.origin.Add(p.Round())
}

func (s *selArea) pointer(gtx layout.Context, e pointer.Event) {
	if e.Kind == pointer.Cancel {
		// Another handler grabbed the pointer - a list scrolling. Gio's
		// cancel event carries no source, so it ends any gesture.
		s.pressing, s.dragging = false, false
		if s.touchSel {
			s.touchSel = false
			s.showCopy = !s.empty()
		}
		return
	}
	if e.Source == pointer.Touch {
		s.touch(gtx, e)
		return
	}
	switch e.Kind {
	case pointer.Press:
		if e.Buttons.Contain(pointer.ButtonSecondary) {
			s.showCopy = !s.empty()
			return
		}
		if !e.Buttons.Contain(pointer.ButtonPrimary) {
			return
		}
		s.showCopy = false
		pos, ok := s.hit(s.abs(e.Position))
		if !ok {
			s.clear()
			return
		}
		gtx.Execute(key.FocusCmd{Tag: &s.tag})
		d := e.Position.Sub(s.lastPos)
		if gtx.Now.Sub(s.lastClick) < 400*time.Millisecond && d.X*d.X+d.Y*d.Y < 25 {
			s.clicks++
		} else {
			s.clicks = 1
		}
		s.lastClick, s.lastPos = gtx.Now, e.Position
		switch {
		case e.Modifiers.Contain(key.ModShift) && s.has:
			s.set(s.anchor, pos)
		case s.clicks == 2:
			s.set(s.word(pos))
		case s.clicks >= 3:
			s.set(selPos{pos.part, 0}, selPos{pos.part, utf8.RuneCountInString(s.prev[pos.part].text)})
		default:
			s.set(pos, pos)
		}
		s.dragging = true
	case pointer.Drag:
		if !s.dragging {
			return
		}
		if pos, ok := s.hit(s.abs(e.Position)); ok {
			s.set(s.anchor, pos)
		}
	case pointer.Release, pointer.Cancel:
		s.dragging = false
	}
}

func (s *selArea) touch(gtx layout.Context, e pointer.Event) {
	switch e.Kind {
	case pointer.Press:
		s.pressing, s.pressAt, s.pressPos, s.pressID = true, gtx.Now, e.Position, e.PointerID
		kit.RequestFrameAt(gtx, gtx.Now.Add(longPress))
	case pointer.Drag:
		if s.touchSel && e.PointerID == s.pressID {
			if pos, ok := s.hit(s.abs(e.Position)); ok {
				s.set(s.anchor, pos)
			}
			return
		}
		d := e.Position.Sub(s.pressPos)
		if slop := float32(gtx.Dp(8)); d.X*d.X+d.Y*d.Y > slop*slop {
			s.pressing = false // a scroll, not a long-press
		}
	case pointer.Release, pointer.Cancel:
		s.pressing = false
		if s.touchSel {
			s.touchSel = false
			s.showCopy = e.Kind == pointer.Release && !s.empty()
		} else if e.Kind == pointer.Release {
			s.clear()
		}
	}
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

func (s *selArea) passFunc() func(gtx layout.Context) bool {
	if s.pass == nil {
		s.pass = s.update
	}
	return s.pass
}

func (e *selectionAreaEl) Layout(gtx layout.Context) layout.Dimensions {
	s := Keyed("giode.selectionarea", e.key, newSelArea)
	s.update(gtx)
	kit.HandleInput(s.passFunc())
	TrackPosition(&s.origin)

	s.seen = s.version
	s.parts = s.parts[:0]
	s.collecting = true
	selAreaStack = append(selAreaStack, s)
	n := len(selAreaStack)
	dims := e.child.Layout(gtx)
	selAreaStack = selAreaStack[:n-1]
	s.collecting = n > 1 && selAreaStack[n-2] == s
	s.prev, s.parts = s.parts, s.prev
	clear(s.parts)
	if s.has {
		// The selection may now be past the end of shorter content.
		for _, p := range []*selPos{&s.anchor, &s.focus} {
			if p.part >= len(s.prev) {
				s.clear()
				break
			}
			p.rune = min(p.rune, utf8.RuneCountInString(s.prev[p.part].text))
		}
	}

	// Watch the pointer over the whole area without taking it from the
	// widgets inside.
	ps := pointer.PassOp{}.Push(gtx.Ops)
	cl := clip.Rect{Max: dims.Size}.Push(gtx.Ops)
	event.Op(gtx.Ops, &s.tag)
	cl.Pop()
	ps.Pop()

	if s.showCopy && !s.empty() {
		s.copy.show(s.bounds(), s.selectedText(), s.clear)
	}
	return dims
}
