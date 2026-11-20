package giode

import (
	"runtime"
	"sync"
	"sync/atomic"

	"gioui.org/app"
	"github.com/xmhrnx/giode/internal/platform/systray"
	"github.com/xmhrnx/giode/internal/platform/window"
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
// Methods may be called from any goroutine, menu and click callbacks
// included, and never wait for the tray: changes reach it a moment
// later, in order. Menu and click callbacks run on the UI goroutine. On Android, iOS and the web the tray does
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
	// dirty is set while a change waits to be applied, and applying
	// while a goroutine applies them.
	dirty    bool
	applying bool
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
	return systray.Supported
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
		systray.Remove()
	}
}

// update starts the tray or applies the current state to it.
func (t *Tray) update() {
	if !systray.Supported {
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
		systray.Start(t.onReady)
		return
	}
	ready := t.ready
	t.mu.Unlock()
	if ready {
		t.schedule()
	}
}

// onReady runs once the platform tray exists.
func (t *Tray) onReady() {
	t.mu.Lock()
	t.ready = true
	t.mu.Unlock()
	t.schedule()
}

// schedule applies the state from a goroutine of the tray's own, so no
// caller waits for it. The platform tray may wait for the main thread
// (fyne.io/systray on macOS does, for every change), and the main
// thread waits for the UI goroutine during a frame: applying there, as
// a SetMenu in a tray checkbox's callback would, deadlocked. Changes
// made while one is applied are applied together after it.
func (t *Tray) schedule() {
	t.mu.Lock()
	t.dirty = true
	start := !t.applying
	t.applying = true
	t.mu.Unlock()
	if start {
		go t.applyPending()
	}
}

// applyPending applies the state until no change is waiting.
func (t *Tray) applyPending() {
	for {
		t.mu.Lock()
		if !t.dirty || t.removed {
			t.applying = false
			t.mu.Unlock()
			return
		}
		t.dirty = false
		t.mu.Unlock()
		t.apply()
	}
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
	trayApplyFunc(t, trayState{icon: icon, template: template, title: title, tooltip: tooltip, items: items, customClick: custom})
}

// clicked runs OnClick on the UI goroutine, or shows or hides the
// window right away: ShowWindow and HideWindow are safe from the
// tray's goroutine, so a hidden window always comes back.
func (t *Tray) clicked() {
	t.mu.Lock()
	fn := t.onClick
	t.mu.Unlock()
	a := t.app
	switch {
	case fn != nil:
		a.Post(fn)
	case a.WindowVisible():
		a.HideWindow()
	default:
		a.ShowWindow()
	}
}

// trayApplyFunc shows a state in the platform tray; tests replace it.
var trayApplyFunc = trayApply

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
// On Linux it works under X11; Wayland can't hide windows (nor bring a
// minimized one back), so there it does nothing, as on Android, iOS
// and the web. It may be called from any goroutine, as may ShowWindow
// and Quit.
func (a *App) HideWindow() {
	if !a.windowHidden.CompareAndSwap(false, true) {
		return
	}
	a.viewMu.Lock()
	early := a.view == nil || !a.view.Valid()
	if early {
		// Before the native window exists (OnReady can run first on
		// Windows): hide it as soon as it does.
		a.hidePending = true
	}
	a.viewMu.Unlock()
	if !early {
		a.hideNow()
	}
}

// hideNow hides the native window for HideWindow, or undoes the hidden
// state where the platform can't hide it, so the tray can always show
// the window again.
func (a *App) hideNow() {
	if !a.hideNativeWindow() {
		a.windowHidden.Store(false)
		return
	}
	a.leftScreen()
}

// ShowWindow shows a window hidden with HideWindow (or by
// MinimizeToTray, CloseToTray), restores it from minimized, and brings
// it to the front.
func (a *App) ShowWindow() {
	a.viewMu.Lock()
	a.hidePending = false
	a.viewMu.Unlock()
	was := a.windowHidden.Swap(false)
	a.showNativeWindow()
	a.invalidate()
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
	return a.closeToTray.Load() && systray.Supported && !a.quitting.Load()
}

// minimizesToTray reports whether minimizing should hide the window.
func (a *App) minimizesToTray() bool {
	return a.minimizeToTray.Load() && systray.Supported
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

// trayApply shows s in the platform tray.
func trayApply(t *Tray, s trayState) {
	var tapped func()
	if s.customClick || runtime.GOOS != "darwin" {
		tapped = t.clicked
	}
	// Otherwise macOS menu bar items open their menu on a click.
	systray.Apply(s.icon, s.template, s.title, s.tooltip, tapped, trayItems(s.items), t.app.Post)
}

func trayItems(items []TrayItem) []systray.Item {
	out := make([]systray.Item, len(items))
	for i, it := range items {
		out[i] = systray.Item{Label: it.Label, Tooltip: it.Tooltip, Checkable: it.Checkable, Checked: it.Checked,
			Disabled: it.Disabled, Separator: it.Separator, Items: trayItems(it.Items), OnClick: it.OnClick}
	}
	return out
}

// --- Native window ----------------------------------------------------------

func (a *App) hideNativeWindow() bool { return window.Hide(a.host()) }
func (a *App) showNativeWindow()      { window.Show(a.host()) }

// raiseNative raises the window through the platform where Gio can't
// (Windows), reporting false where it goes through Gio.
func (a *App) raiseNative() bool { return window.Raise(a.host()) }

// platformInit sets the window up before the app finishes launching.
func (a *App) platformInit() {
	if a.cfg.MenuBarApp {
		window.MenuBarApp()
	}
	window.Init(a.host(), a.cfg.Transparent, func() bool {
		if a.vetoClose() {
			return false
		}
		if a.closesToTray() {
			a.HideWindow()
			return false
		}
		return true
	})
	if launchHook != nil {
		launchHook(a)
	}
}

// launchHook, where set, runs at platformInit (macOS's Services).
var launchHook func(*App)

// windowHook, where set, runs when a tray option turns on and the
// window exists (Windows subclasses the window for them).
var windowHook func(*App)

// trayView sets up closing and Dock re-opening once the window exists.
func (a *App) trayView(app.ViewEvent) {
	window.Shown(a.host(), a.closeToTray.Load() || a.cfg.OnCloseRequest != nil || a.cfg.NoClose)
}

// applyTrayOptions applies CloseToTray and MinimizeToTray once turned on.
func (a *App) applyTrayOptions() {
	if a.closeToTray.Load() {
		window.InterceptClose(a.host())
	}
	if windowHook != nil {
		windowHook(a)
	}
	a.warnTrayLimits()
}

// trayCloseWarned limits the CloseToTray warning to once.
var trayCloseWarned atomic.Bool

func (a *App) warnTrayLimits() {
	if a.closeToTray.Load() && window.CloseNeedsPatch && trayCloseWarned.CompareAndSwap(false, true) {
		a.logf(LogLevelWarning, "giode: CloseToTray on Linux needs giode build's Gio patches; closing the window quits")
	}
}

func init() {
	window.Hooks.VetoClose = func() bool {
		a := currentApp.Load()
		return a != nil && a.vetoClose()
	}
	window.Hooks.HideOnClose = func() bool {
		a := currentApp.Load()
		return a != nil && a.closesToTray()
	}
	window.Hooks.HiddenOnClose = func() {
		if a := currentApp.Load(); a != nil {
			a.windowHiddenNatively()
		}
	}
	window.Hooks.Reopen = func() {
		if a := currentApp.Load(); a != nil && a.windowHidden.Load() {
			a.ShowWindow()
		}
	}
}
