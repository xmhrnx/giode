// Package scroll provides a stateful vertical scroll container.

package scroll

// ContentHeight returns the total height of the scrolled content, as
// measured on the last frame.
func (s *Scroll) ContentHeight() int {
	return s.contentH
}

// ViewportHeight returns the height of the visible viewport, as
// measured on the last frame.
func (s *Scroll) ViewportHeight() int {
	return s.viewportH
}

// Progress returns how far scrolled, from 0 (top) to 1 (bottom); 0
// when there's nothing to scroll (the content fits the viewport).
func (s *Scroll) Progress() float32 {
	max := s.maxOffset()
	if max <= 0 {
		return 0
	}
	return s.offset / float32(max)
}

// AtBottom reports whether the scroll is at (or past) its maximum
// offset - also true when there's nothing to scroll at all (the
// content already fits the viewport), the same way a page with no
// scrollbar is trivially "at the bottom": there's nowhere else to go,
// so a load-more handler driven by AtBottom/OnReachBottom fires
// immediately, prompting an initial fetch when the first page of
// content doesn't even fill the viewport.
func (s *Scroll) AtBottom() bool {
	return s.offset >= float32(s.maxOffset())
}

// Visible reports whether the vertical range [top, top+height) - in
// the same coordinate space as Content's own output, e.g. a running
// offset tracked while building a long list - is at least partially
// within the current viewport. Useful for lazy-loading or skipping
// expensive work for an item that isn't currently on screen.
func (s *Scroll) Visible(top, height int) bool {
	return top+height > int(s.offset) && top < int(s.offset)+s.viewportH
}

// VisibleDp is Visible with top and height in dp, the units Styles
// use, converted with the display scale of the last frame.
func (s *Scroll) VisibleDp(top, height float32) bool {
	scale := s.pxPerDp
	if scale <= 0 {
		scale = 1
	}
	return s.Visible(int(top*scale), int(height*scale+0.5))
}

// OnReachBottom sets the handler invoked once when the scroll reaches
// its maximum offset - edge-triggered: it fires again only after
// scrolling away from the bottom and back, not on every frame spent
// there. It returns s for chaining.
func (s *Scroll) OnReachBottom(fn func()) *Scroll {
	s.onReachBottom = fn
	return s
}
