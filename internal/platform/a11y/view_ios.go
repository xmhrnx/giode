//go:build ios

package a11y

import "gioui.org/app"

// The view gets Gio's view, under the view controller, its
// accessibility elements.
// ViewReady gives the native view its accessibility elements.
func ViewReady(a Host, ev app.ViewEvent) {
	if v, ok := ev.(app.UIKitViewEvent); ok {
		install(a, v.ViewController)
	}
}
