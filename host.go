package giode

import (
	"context"
	"image"
	"time"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/io/pointer"

	"github.com/xmhrnx/giode/internal/platform"
	"github.com/xmhrnx/giode/internal/platform/a11y"
)

// appHost gives the platform packages what they need of the App.
type appHost struct{ a *App }

func (a *App) host() platform.Host { return appHost{a} }

func (h appHost) NativeHandle() uintptr     { return h.a.NativeHandle() }
func (h appHost) NativeView() app.ViewEvent { return h.a.NativeView() }
func (h appHost) Post(f func())             { h.a.Post(f) }
func (h appHost) Window() *app.Window       { return h.a.window }
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

// a11yHost gives the accessibility bridges what they need of the App.
type a11yHost struct{ appHost }

func (a *App) a11yHost() a11y.Host { return a11yHost{appHost{a}} }

func (h a11yHost) Invalidate()   { h.a.invalidate() }
func (h a11yHost) Focused() bool { return h.a.Focused() }

func (h a11yHost) Size() (image.Point, bool) {
	f, err := h.a.Frame()
	if err != nil {
		return image.Point{}, false
	}
	s := h.a.scale()
	return image.Pt(int(f.Width*s), int(f.Height*s)), true
}

// Tap taps the middle of r, as a finger would, for a node with no
// component to call. It needs giode's Gio patches.
func (h a11yHost) Tap(r image.Rectangle) bool {
	c := r.Min.Add(r.Size().Div(2))
	p := f32.Pt(float32(c.X), float32(c.Y))
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		for _, k := range []pointer.Kind{pointer.Press, pointer.Release} {
			if injectEvent(ctx, h.a, pointer.Event{Kind: k, Source: pointer.Touch, Position: p, Time: time.Since(inputEpoch)}) != nil {
				return
			}
		}
	}()
	return true
}
