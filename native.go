package giode

import (
	"reflect"

	"gioui.org/app"
)

// handleViewEvent records the platform view and applies what needs
// the native window: the icon and file drop on Windows.
func (a *App) handleViewEvent(ev app.ViewEvent) {
	a.viewMu.Lock()
	a.view = ev
	pending := a.hidePending && ev.Valid()
	if pending {
		a.hidePending = false
	}
	a.viewMu.Unlock()
	a.platformView(ev)
	a.trayView(ev)
	a.placementView(ev)
	a.startInTray(ev)
	if pending && a.windowHidden.Load() {
		a.hideNow()
	}
}

// NativeView returns the platform view event Gio reported for the
// window: an app.Win32ViewEvent on Windows, app.AppKitViewEvent on
// macOS, app.X11ViewEvent or app.WaylandViewEvent on Linux, ... It is
// nil until the window exists. Use it for platform integrations Giode
// doesn't cover.
func (a *App) NativeView() app.ViewEvent {
	a.viewMu.Lock()
	defer a.viewMu.Unlock()
	return a.view
}

// NativeHandle returns the native window handle: the HWND on Windows,
// the NSView on macOS, the X11 window ID or the wl_surface pointer on
// Linux, the android.view.View reference on Android. It is 0 until the
// window exists.
func (a *App) NativeHandle() uintptr {
	ev := a.NativeView()
	if ev == nil || !ev.Valid() {
		return 0
	}
	// The concrete event types only exist on their own platforms, so
	// they're read by field name.
	v := reflect.ValueOf(ev)
	for v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return 0
	}
	for _, name := range []string{"HWND", "View", "Window", "Surface", "ViewController"} {
		f := v.FieldByName(name)
		if !f.IsValid() {
			continue
		}
		switch f.Kind() {
		case reflect.Uintptr, reflect.Uint, reflect.Uint64, reflect.Uint32:
			return uintptr(f.Uint())
		case reflect.UnsafePointer:
			return uintptr(f.Pointer())
		}
	}
	return 0
}
