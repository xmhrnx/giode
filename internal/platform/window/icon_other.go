//go:build !windows

package window

import (
	"image"

	"gioui.org/app"
)

// SetIcon: other platforms have no runtime window icon (it comes from
// the application bundle there).
func SetIcon(view app.ViewEvent, img image.Image) bool { return false }
