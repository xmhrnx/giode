//go:build linux && !android && nox11

package window

import "gioui.org/app"

func hideX11Window(app.ViewEvent) bool { return false }
func showX11Window(app.ViewEvent)      {}
