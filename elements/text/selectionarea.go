package text

import (
	"unicode/utf8"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements/internal/base"
)

type selectionAreaEl struct {
	key   any
	child base.Element
}

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
func SelectionArea(key any, child base.Element) base.Element {
	return &selectionAreaEl{key: key, child: child}
}

func (e *selectionAreaEl) Layout(gtx layout.Context) layout.Dimensions {
	s := base.Keyed("giode.selectionarea", e.key, newSelArea)
	s.update(gtx)
	kit.HandleInput(s.passFunc())
	base.TrackPosition(&s.origin)

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
