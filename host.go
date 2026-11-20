package giode

import (
	"context"

	"gioui.org/app"

	"github.com/xmhrnx/giode/internal/platform"
)

// appHost gives the platform packages what they need of the App.
type appHost struct{ a *App }

func (a *App) host() platform.Host { return appHost{a} }

func (h appHost) NativeHandle() uintptr     { return h.a.NativeHandle() }
func (h appHost) NativeView() app.ViewEvent { return h.a.NativeView() }
func (h appHost) Post(f func())             { h.a.Post(f) }
func (h appHost) RunOnWindow(f func())      { h.a.window.Run(f) }
func (h appHost) Title() string             { return h.a.Title() }
func (h appHost) Logf(level platform.Level, format string, args ...any) {
	h.a.logf(LogLevel(level), format, args...)
}

// onAndroidMain runs fn with the GioView on the Android main thread
// and waits for it.
func (a *App) onAndroidMain(ctx context.Context, fn func(view uintptr) error) error {
	return platform.OnAndroidMain(ctx, a.host(), fn)
}
