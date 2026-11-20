//go:build darwin && !ios

package a11y

import "gioui.org/app"

// The view gets Gio's NSView its accessibility elements.
// ViewReady gives the native view its accessibility elements.
func ViewReady(a Host, ev app.ViewEvent) {
	if v, ok := ev.(app.AppKitViewEvent); ok {
		install(a, v.View)
	}
}
