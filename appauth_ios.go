//go:build ios

package giode

import "gioui.org/app"

// authAnchorView is the UIViewController a sign-in sheet shows over.
func (a *App) authAnchorView() uintptr {
	if v, ok := a.NativeView().(app.UIKitViewEvent); ok {
		return v.ViewController
	}
	return 0
}
