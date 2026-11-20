//go:build !darwin

package a11y

import "gioui.org/app"

// Nothing to set up on the view here: the bridge, if any,
// starts elsewhere.
// ViewReady gives the native view its accessibility elements.
func ViewReady(a Host, ev app.ViewEvent) {}
