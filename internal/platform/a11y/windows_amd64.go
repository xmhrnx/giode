//go:build windows && amd64

package a11y

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// elementFromPointCallback is the fragment root's
// ElementProviderFromPoint. Its point arrives as two doubles in
// floating-point registers, which a Go callback can't read on 64-bit
// Windows; screen readers ask for the point under the pointer (Narrator
// following the mouse, Inspect's hover), so it answers for the cursor's
// position. The out parameter is in the fourth integer register: amd64
// numbers registers by argument position.
func elementFromPointCallback() uintptr {
	return windows.NewCallback(func(this unsafe.Pointer, _, _ uintptr, out *unsafe.Pointer) uintptr {
		return fromCursor(this, out)
	})
}

// uiaRangeFromPointCallback is ITextProvider.RangeFromPoint, whose
// UiaPoint (two doubles, 16 bytes) amd64 passes by reference.
func uiaRangeFromPointCallback(f func(this unsafe.Pointer, out *unsafe.Pointer) uintptr) uintptr {
	return windows.NewCallback(func(this unsafe.Pointer, point uintptr, out *unsafe.Pointer) uintptr {
		return f(this, out)
	})
}
