//go:build darwin && !ios

package giode

import "gioui.org/app"

// authAnchorView is the NSView a sign-in sheet shows over.
func (a *App) authAnchorView() uintptr {
	if v, ok := a.NativeView().(app.AppKitViewEvent); ok {
		return v.View
	}
	return 0
}
