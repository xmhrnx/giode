package giode

import "github.com/xmhrnx/giode/internal/platform"

// appHost gives the platform packages what they need of the App.
type appHost struct{ a *App }

func (a *App) host() platform.Host { return appHost{a} }

func (h appHost) NativeHandle() uintptr { return h.a.NativeHandle() }
func (h appHost) Post(f func())         { h.a.Post(f) }
func (h appHost) RunOnWindow(f func())  { h.a.window.Run(f) }
func (h appHost) Title() string         { return h.a.Title() }
func (h appHost) Logf(level platform.Level, format string, args ...any) {
	h.a.logf(LogLevel(level), format, args...)
}
