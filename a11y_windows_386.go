//go:build windows && 386

package giode

import (
	"math"
	"unsafe"

	"golang.org/x/sys/windows"
)

// elementFromPointCallback is the fragment root's
// ElementProviderFromPoint: on 32-bit Windows the point's two doubles
// arrive on the stack, two words each.
func elementFromPointCallback() uintptr {
	return windows.NewCallback(func(this unsafe.Pointer, x0, x1, y0, y1 uintptr, out *unsafe.Pointer) uintptr {
		x := math.Float64frombits(uint64(x0) | uint64(x1)<<32)
		y := math.Float64frombits(uint64(y0) | uint64(y1)<<32)
		*out = nodeOf(this, offRoot).b.fromScreen(int32(x), int32(y))
		return sOK
	})
}

// uiaRangeFromPointCallback is ITextProvider.RangeFromPoint, whose
// UiaPoint 32-bit Windows passes on the stack: four words.
func uiaRangeFromPointCallback(f func(this unsafe.Pointer, out *unsafe.Pointer) uintptr) uintptr {
	return windows.NewCallback(func(this unsafe.Pointer, x0, x1, y0, y1 uintptr, out *unsafe.Pointer) uintptr {
		return f(this, out)
	})
}
