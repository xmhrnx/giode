//go:build ios

package signin

import (
	"gioui.org/app"
	"github.com/xmhrnx/giode/internal/platform"
)

// anchorView is the UIViewController a sign-in sheet shows over.
func anchorView(h platform.Host) uintptr {
	if v, ok := h.NativeView().(app.UIKitViewEvent); ok {
		return v.ViewController
	}
	return 0
}
