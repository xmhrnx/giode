//go:build windows && arm64

package a11y

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// elementFromPointCallback is the fragment root's
// ElementProviderFromPoint. Its point arrives as two doubles in
// floating-point registers a Go callback can't read; it answers for the
// cursor's position (see the amd64 version). On arm64 the out parameter
// is the second integer register.
func elementFromPointCallback() uintptr {
	return windows.NewCallback(func(this unsafe.Pointer, out *unsafe.Pointer) uintptr {
		return fromCursor(this, out)
	})
}

// uiaRangeFromPointCallback is ITextProvider.RangeFromPoint, whose
// UiaPoint arm64 passes in two floating-point registers.
func uiaRangeFromPointCallback(f func(this unsafe.Pointer, out *unsafe.Pointer) uintptr) uintptr {
	return windows.NewCallback(func(this unsafe.Pointer, out *unsafe.Pointer) uintptr {
		return f(this, out)
	})
}
