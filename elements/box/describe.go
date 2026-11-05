package box

import (
	"strconv"

	"github.com/xmhrnx/giode/elements/internal/base"
)

// Describe names the box for diagnostics: its kind and the first text
// inside it, which is usually enough to find it in the code.
func (b *BoxElement) Describe() string {
	kind := "Box"
	if s := firstText(b, 4); s != "" {
		return kind + " containing " + strconv.Quote(s)
	}
	return kind
}

// firstText returns the text of the first Text element within depth
// levels of b.
func firstText(b *BoxElement, depth int) string {
	if depth == 0 {
		return ""
	}
	for _, c := range b.children {
		if t, ok := c.(base.TextInliner); ok {
			return t.InlineText()
		}
		if cb, ok := c.(*BoxElement); ok {
			if s := firstText(cb, depth-1); s != "" {
				return s
			}
		}
	}
	return ""
}
