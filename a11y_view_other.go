//go:build !darwin

package giode

import "gioui.org/app"

// a11yView has nothing to set up on the view here: the bridge, if any,
// starts elsewhere.
func (a *App) a11yView(ev app.ViewEvent) {}
