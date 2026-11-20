//go:build darwin && !ios

package signin

import (
	"gioui.org/app"
	"github.com/xmhrnx/giode/internal/platform"
)

// anchorView is the NSView a sign-in sheet shows over.
func anchorView(h platform.Host) uintptr {
	if v, ok := h.NativeView().(app.AppKitViewEvent); ok {
		return v.View
	}
	return 0
}
