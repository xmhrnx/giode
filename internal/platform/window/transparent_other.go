//go:build !windows && !(darwin && !ios)

package window

import "github.com/xmhrnx/giode/internal/platform"

// Linux sets transparency through giode's Gio patches before the window
// exists (window_linux_gio8.go); phones and the web don't have it.
// MakeTransparent lets the window show through where the UI is
// transparent.
func MakeTransparent(h platform.Host) {}
