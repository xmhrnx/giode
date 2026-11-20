//go:build linux && !android && nowayland

package window

import "gioui.org/app"

func isWayland(app.ViewEvent) bool { return false }
