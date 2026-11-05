package box

// --- Box sugar -----------------------------------------------------------------

// OnClick makes the box clickable, as if wrapped in a Pressable: the
// pointer cursor shows on hover, Enter/Space activate it when
// focused, and clickable children keep their own clicks. Its text stays
// selectable, like a clickable element on the web: in a SelectionArea,
// or Selectable text, a mouse drag selects instead of clicking. Like OnHover,
// it needs a box created once and reused every frame (or looked up
// with Keyed), since press state lives in the element.
func (b *BoxElement) OnClick(fn func(e ClickEvent)) *BoxElement {
	if b.press == nil {
		b.press = NewPressable(nil)
		b.press.selectable = true
	}
	b.press.OnClick(fn)
	return b
}

// Pressable returns the Pressable behind OnClick, to set a label,
// double-click handlers or disable it; nil before OnClick.
func (b *BoxElement) Pressable() *Pressable {
	return b.press
}
