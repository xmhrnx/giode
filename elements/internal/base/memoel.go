package base

import "gioui.org/layout"

type memoEl struct {
	key   any
	build func() Element
	deps  []any
}

// Memo returns an element that caches build's output under key and
// rebuilds it only when deps, the States it reads, its constraints or
// the theme change, or when a widget inside asks for it - see
// MemoCache for the full rules. Use it inside the view function to
// skip rebuilding parts of the UI that didn't change:
//
//	giode.Memo("sidebar", func() giode.Element {
//		return sidebar(user.Name)
//	}, user.Name)
//
// key identifies the cache across frames and must be unique among the
// memos laid out in a frame (any comparable value: a string, an ID, a
// pointer). A key not laid out for a frame is released.
func Memo(key any, build func() Element, deps ...any) Element {
	return &memoEl{key: key, build: build, deps: deps}
}

func (m *memoEl) Layout(gtx layout.Context) layout.Dimensions {
	c, ok := memos[m.key]
	if !ok {
		c = new(MemoCache)
		memos[m.key] = c
	}
	return c.Layout(gtx, m.build, m.deps...)
}
