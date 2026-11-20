//go:build darwin && !ios

package giode

import "gioui.org/app"

// a11yView gives Gio's NSView its accessibility elements.
func (a *App) a11yView(ev app.ViewEvent) {
	if v, ok := ev.(app.AppKitViewEvent); ok {
		a.a11yInstall(v.View)
	}
}
