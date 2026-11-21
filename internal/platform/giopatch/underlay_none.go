//go:build !giode_gio && !giode_gio2

package giopatch

// Target is a frame's GPU target, as Gio's patches report it.
type Target struct {
	API           int
	Framebuffer   uint32
	Handle        uintptr
	Width, Height int
}

// SetUnderlay needs giode's Gio patches.
func SetUnderlay(draw, after func(Target), release func(), forceGL bool) error {
	return ErrUnderlayUnpatched
}
