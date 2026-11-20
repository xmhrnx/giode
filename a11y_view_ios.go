//go:build ios

package giode

import "gioui.org/app"

// a11yView gives Gio's view, under the view controller, its
// accessibility elements.
func (a *App) a11yView(ev app.ViewEvent) {
	if v, ok := ev.(app.UIKitViewEvent); ok {
		a.a11yInstall(v.ViewController)
	}
}
