//go:build android || ios || (!windows && !linux && !darwin)

package filedrag

import "gioui.org/app"

func Install() {}

func ViewReady(view app.ViewEvent) string { return "aren't supported on this platform" }
