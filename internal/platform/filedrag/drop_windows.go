package filedrag

import "gioui.org/app"

// Install: the app's window procedure takes drops once the window
// exists (AcceptFiles, RegisterOLE).
func Install() {}

func ViewReady(view app.ViewEvent) string { return "" }
