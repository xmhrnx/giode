package giode

import (
	"sync"

	"gioui.org/app"
)

// Tray is the app's icon in the system tray: the notification area on
// Windows, the menu bar on macOS (where it can show text too) and the
// status notifier area on Linux desktops that have one (KDE, most
// others with an extension; GNOME needs the AppIndicator extension).
// Get it with App.Tray; it appears once something is set on it:
//
//	app.Tray().
//		SetIcon(iconPNG).
//		SetTooltip("Pocket Agent").
//		SetMenu(
//			giode.TrayAction("Open", app.ShowWindow),
//			giode.TraySeparator(),
//			giode.TrayAction("Quit", app.Quit),
//		)
//
// On Windows and Linux, clicking the icon shows or hides the window
// unless OnClick says otherwise, and the menu opens with a right-click.
// On macOS, as menu bar items do, a click opens the menu (or runs
// OnClick when set). Config.MinimizeToTray and Config.CloseToTray hide
// the window to the tray instead of minimizing or closing it.
//
// Methods may be called from any goroutine; menu and click callbacks
// run on the UI goroutine. On Android, iOS and the web the tray does
// nothing: TraySupported reports false.
type Tray struct {
	app *App

	mu       sync.Mutex
	icon     []byte
	template bool
	title    string
	tooltip  string
	items    []TrayItem
	onClick  func()
	started  bool
	ready    bool
	removed  bool
}

// TrayItem is an entry of the tray's menu. Build them with TrayAction,
// TrayCheckbox, TraySeparator and TraySubmenu, or fill in the fields.
type TrayItem struct {
	// Label is the item's text.
	Label string
	// Tooltip is shown on hover where the platform supports it.
	Tooltip string
	// Disabled greys the item out.
	Disabled bool
	// Checkable shows a check mark when Checked.
	Checkable, Checked bool
	// Separator makes the item a divider line; the other fields are
	// ignored.
	Separator bool
	// Items makes the item a submenu.
	Items []TrayItem
	// OnClick runs on the UI goroutine when the item is chosen.
	OnClick func()
}

// TrayAction returns a menu item running fn.
func TrayAction(label string, fn func()) TrayItem {
	return TrayItem{Label: label, OnClick: fn}
}

// TrayCheckbox returns a menu item with a check mark when checked. It
// doesn't toggle itself: fn updates your state, and the next SetMenu
// shows it.
func TrayCheckbox(label string, checked bool, fn func()) TrayItem {
	return TrayItem{Label: label, Checkable: true, Checked: checked, OnClick: fn}
}

// TraySeparator returns a divider line.
func TraySeparator() TrayItem {
	return TrayItem{Separator: true}
}

// TraySubmenu returns a submenu of items.
func TraySubmenu(label string, items ...TrayItem) TrayItem {
	return TrayItem{Label: label, Items: items}
}

// TraySupported reports whether this platform has a system tray giode
// can use: Windows, macOS and Linux.
func TraySupported() bool {
	return traySupported
}

// Tray returns the app's tray icon, created hidden on first use.
func (a *App) Tray() *Tray {
	a.trayOnce.Do(func() { a.tray = &Tray{app: a} })
	return a.tray
}

// SetIcon sets the icon from PNG bytes. Without one the tray uses
// Config.Icon. Small icons work best: 32x32 or 64x64.
func (t *Tray) SetIcon(png []byte) *Tray {
	t.mu.Lock()
	t.icon, t.template = png, false
	t.mu.Unlock()
	t.update()
	return t
}

// SetTemplateIcon sets an icon that macOS tints to match the menu bar,
// light or dark: PNG bytes, black shapes on transparency. Elsewhere it
// is a plain icon.
func (t *Tray) SetTemplateIcon(png []byte) *Tray {
	t.mu.Lock()
	t.icon, t.template = png, true
	t.mu.Unlock()
	t.update()
	return t
}

// SetTitle sets text shown next to the icon: in the macOS menu bar, and
// on Linux panels that show it. Use it for live information - an
// unread count, a timer, a status. Windows doesn't show it; set the
// tooltip there.
func (t *Tray) SetTitle(title string) *Tray {
	t.mu.Lock()
	t.title = title
	t.mu.Unlock()
	t.update()
	return t
}

// SetTooltip sets the text shown when the pointer rests on the icon.
func (t *Tray) SetTooltip(tooltip string) *Tray {
	t.mu.Lock()
	t.tooltip = tooltip
	t.mu.Unlock()
	t.update()
	return t
}

// SetMenu replaces the menu. Call it again to change items or check
// marks.
func (t *Tray) SetMenu(items ...TrayItem) *Tray {
	t.mu.Lock()
	t.items = items
	t.mu.Unlock()
	t.update()
	return t
}

// OnClick sets what a click on the icon does, on the UI goroutine. By
// default it shows the window, or hides it when it's showing - except
// on macOS, where a click opens the menu unless OnClick is set.
func (t *Tray) OnClick(fn func()) *Tray {
	t.mu.Lock()
	t.onClick = fn
	t.mu.Unlock()
	t.update()
	return t
}

// Remove takes the icon out of the tray for good.
func (t *Tray) Remove() {
	t.mu.Lock()
	started := t.started && !t.removed
	t.removed = true
	t.mu.Unlock()
	if started {
		trayRemove()
	}
}

// update starts the tray or applies the current state to it.
func (t *Tray) update() {
	if !traySupported {
		return
	}
	t.mu.Lock()
	if t.removed {
		t.mu.Unlock()
		return
	}
	if !t.started {
		t.started = true
		t.mu.Unlock()
		trayStart(t)
		return
	}
	ready := t.ready
	t.mu.Unlock()
	if ready {
		t.apply()
	}
}

// onReady runs once the platform tray exists.
func (t *Tray) onReady() {
	t.mu.Lock()
	t.ready = true
	t.mu.Unlock()
	t.apply()
}

// apply pushes the whole state to the platform tray.
func (t *Tray) apply() {
	t.mu.Lock()
	icon, template, title, tooltip, items := t.icon, t.template, t.title, t.tooltip, t.items
	custom := t.onClick != nil
	t.mu.Unlock()
	if icon == nil {
		icon = t.app.cfg.Icon
	}
	trayApply(t, trayState{icon: icon, template: template, title: title, tooltip: tooltip, items: items, customClick: custom})
}

// clicked runs the click action on the UI goroutine.
func (t *Tray) clicked() {
	t.mu.Lock()
	fn := t.onClick
	t.mu.Unlock()
	a := t.app
	a.Post(func() {
		if fn != nil {
			fn()
			return
		}
		if a.WindowVisible() {
			a.HideWindow()
		} else {
			a.ShowWindow()
		}
	})
}

// trayState is what the platform tray shows.
type trayState struct {
	icon     []byte
	template bool
	title    string
	tooltip  string
	items    []TrayItem
	// customClick is set when OnClick was given.
	customClick bool
}

// --- Window visibility ------------------------------------------------------

// HideWindow hides the window without closing it: it leaves the screen
// and the taskbar or Dock, and ShowWindow brings it back as it was.
// With a tray icon it's how an app keeps running in the background.
// On Linux it works under X11; Wayland can't hide windows, so there it
// minimizes. On Android, iOS and the web it does nothing.
func (a *App) HideWindow() {
	if a.windowHidden.Swap(true) {
		return
	}
	a.hideNativeWindow()
	a.leftScreen()
}

// ShowWindow shows a window hidden with HideWindow (or by
// MinimizeToTray, CloseToTray), restores it from minimized, and brings
// it to the front.
func (a *App) ShowWindow() {
	was := a.windowHidden.Swap(false)
	a.showNativeWindow()
	a.window.Invalidate()
	if was && a.cfg.OnForeground != nil {
		a.Post(a.cfg.OnForeground)
	}
}

// WindowVisible reports whether the window is on screen: not hidden
// and not minimized.
func (a *App) WindowVisible() bool {
	return !a.windowHidden.Load() && a.WindowState() != WindowMinimized
}

// Quit closes the window and ends the app, even with CloseToTray.
func (a *App) Quit() {
	a.quitting.Store(true)
	if a.windowHidden.Load() {
		a.showNativeWindow()
	}
	a.Close()
}

// closesToTray reports whether a close request should hide the window
// instead.
func (a *App) closesToTray() bool {
	return a.closeToTray.Load() && traySupported && !a.quitting.Load()
}

// minimizesToTray reports whether minimizing should hide the window.
func (a *App) minimizesToTray() bool {
	return a.minimizeToTray.Load() && traySupported
}

// SetCloseToTray turns Config.CloseToTray on or off while the app
// runs: closing the window then hides it to the tray, or ends the app.
// It may be called from any goroutine.
func (a *App) SetCloseToTray(on bool) {
	if a.closeToTray.Swap(on) != on && on {
		a.applyTrayOptions()
	}
}

// CloseToTray reports whether closing the window hides it to the tray.
func (a *App) CloseToTray() bool {
	return a.closeToTray.Load()
}

// SetMinimizeToTray turns Config.MinimizeToTray on or off while the
// app runs. It may be called from any goroutine.
func (a *App) SetMinimizeToTray(on bool) {
	if a.minimizeToTray.Swap(on) != on && on {
		a.applyTrayOptions()
	}
}

// MinimizeToTray reports whether minimizing the window hides it to the
// tray.
func (a *App) MinimizeToTray() bool {
	return a.minimizeToTray.Load()
}

// startInTray hides the window as soon as it exists when this launch
// starts in the tray (Config.StartInTray).
func (a *App) startInTray(ev app.ViewEvent) {
	if a.startedInTray || !ev.Valid() {
		return
	}
	a.startedInTray = true
	if a.cfg.StartInTray.startsInTray() {
		a.HideWindow()
	}
}

// windowHiddenNatively records that the platform hid the window
// instead of closing it.
func (a *App) windowHiddenNatively() {
	if !a.windowHidden.Swap(true) {
		a.leftScreen()
	}
}

// leftScreen runs OnBackground for a window hidden while not minimized
// (minimizing already ran it).
func (a *App) leftScreen() {
	if a.WindowState() != WindowMinimized && a.cfg.OnBackground != nil {
		a.Post(a.cfg.OnBackground)
	}
}
