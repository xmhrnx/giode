package platform

import (
	"context"

	"gioui.org/app"
)

// Level is a log message's level, as giode.LogLevel.
type Level int

const (
	Debug Level = iota
	Info
	Warning
	Error
)

// Host is what platform code needs of the app: its window and its UI
// goroutine. The root package implements it for giode.App.
type Host interface {
	// NativeHandle is the window's native handle (giode.App.NativeHandle):
	// an HWND, an NSView, a GioView..., or 0 before the window exists.
	NativeHandle() uintptr
	// NativeView is the window's view event (giode.App.NativeView), or
	// nil.
	NativeView() app.ViewEvent
	// Window is the app's Gio window.
	Window() *app.Window
	// Post runs f on the UI goroutine.
	Post(f func())
	// RunOnWindow runs f on the window's own thread (Gio's Window.Run)
	// and waits for it.
	RunOnWindow(f func())
	// Logf logs through the app's Logger.
	Logf(level Level, format string, args ...any)
	// Title is the app's title.
	Title() string
}

// OnAndroidMain runs fn with the GioView on the Android main thread and
// waits for it, or for ctx.
func OnAndroidMain(ctx context.Context, h Host, fn func(view uintptr) error) error {
	view := h.NativeHandle()
	if view == 0 {
		return ErrNotSupported
	}
	done := make(chan error, 1)
	go func() {
		var err error
		h.RunOnWindow(func() { err = fn(view) })
		done <- err
	}()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
