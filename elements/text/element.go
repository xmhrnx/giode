package text

import (
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/styles"
)

// TextElement renders a single string with its styles. Chain OnHover
// to add hover styles.
type TextElement struct {
	str string
	st  styles.Styles

	kit.HoverTracker
	hoverSt     styles.Styles
	lastHovered bool

	// selKey identifies a selectable text's state; see Selectable.
	selKey any
}

// Text renders a string with the given styles. The styles argument is
// optional.
func Text(str string, st ...styles.Styles) *TextElement {
	var s styles.Styles
	if len(st) > 0 {
		s = st[0]
	}
	return &TextElement{str: str, st: s}
}

// OnHover sets styles applied while the pointer is over the text,
// merged on top of its base styles - color, decoration, weight,
// anything Styles' text properties support:
//
//	label := giode.Text("Docs", giode.Styles{Color: "#94a3b8"}).OnHover(giode.Styles{
//		Color:          "#f8fafc",
//		TextDecoration: giode.TextDecorationUnderline,
//		Cursor:         giode.CursorPointer,
//	})
//
// Hover state lives in the element, so create it once and reuse it
// every frame; rebuilding it in the view function resets the hover on
// each frame.
func (t *TextElement) OnHover(st styles.Styles) *TextElement {
	t.Enable()
	t.hoverSt = st
	return t
}

// CacheSafe reports true: the element depends only on its fields, which
// don't change once it's built, and on what caches already watch (the
// theme, the selection, its input passes).
func (*TextElement) CacheSafe() bool { return true }
