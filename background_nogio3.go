//go:build !giode_gio3

package giode

import "image/color"

// setNativeBackground needs the giode CLI's Gio patches; plain Gio shows
// the window white until its first frame on Windows.
func setNativeBackground(color.NRGBA, bool) {}
