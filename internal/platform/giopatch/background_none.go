//go:build !giode_gio3

package giopatch

import "image/color"

// SetBackground needs the giode CLI's Gio patches; plain Gio shows
// the window white until its first frame on Windows.
func SetBackground(color.NRGBA, bool) {}
