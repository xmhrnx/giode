package giode

import (
	"image"
	"image/color"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/unit"
	"gioui.org/x/explorer"
	"github.com/xmhrnx/giode/components/menu"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/host"
	"github.com/xmhrnx/giode/i18n"
	"github.com/xmhrnx/giode/state"
	"github.com/xmhrnx/giode/styles/properties"
)

// App owns a window and its render loop.
type App struct {
	cfg Config
	// mcpBound is the MCP server's listen address once it listens.
	mcpBound   atomic.Pointer[string]
	window     *app.Window
	winPointer windowPointer
	ctxMenu    *menu.Menu

	icon        image.Image
	iconApplied bool

	instance       *host.InstanceLock
	rendererErr    error
	secondInstance bool

	winState   atomic.Int32
	winFocused atomic.Bool
	// The last frame's size and scale, for Screen and the window
	// position in dp.
	frameGeom atomic.Pointer[frameGeom]
	topMost   atomic.Bool
	title     atomic.Pointer[string]

	clip clipboardState

	viewMu sync.Mutex
	view   app.ViewEvent
	// hidePending is a HideWindow called before the native window
	// existed (Gio can draw the first frame before reporting it),
	// applied when it does. Guarded by viewMu.
	hidePending bool
	dropWarned  bool
	drag        fileDragState

	// The system bar colors last applied; see updateSystemBars.
	barsApplied bool
	statusColor color.NRGBA
	navColor    color.NRGBA

	hideKeyboard atomic.Bool
	frames       frameRecorder
	// maxFPS is the frame-rate cap (Config.MaxFPS, SetMaxFPS); 0 for
	// none. lastFrame is when the last frame started.
	maxFPS    atomic.Int64
	lastFrame time.Time
	themeCheck   chan struct{}

	// explorer shows the file dialogs; it sees every window event.
	explorer *explorer.Explorer

	mcp     *mcpBridge
	uiMu    sync.Mutex
	uiCalls []func()

	// The tray icon (see Tray), and whether the window is hidden to it.
	tray         *Tray
	trayOnce     sync.Once
	windowHidden atomic.Bool
	quitting     atomic.Bool
	// The tray options, from Config and the setters.
	closeToTray    atomic.Bool
	minimizeToTray atomic.Bool
	startedInTray  bool
}

// New creates an App from cfg. The window is created and shown the
// first time the app runs.
func New(cfg Config) *App {
	if cfg.Size == (image.Point{}) {
		cfg.Size = image.Pt(800, 600)
	}
	if properties.CalcColor(cfg.Background).A == 0 {
		cfg.Background = "#000000"
	}
	// Per-axis size bounds combine with the point bounds; the tighter
	// bound wins.
	if cfg.MinWidth > 0 && (cfg.MinSize.X == 0 || cfg.MinSize.X < cfg.MinWidth) {
		cfg.MinSize.X = cfg.MinWidth
	}
	if cfg.MinHeight > 0 && (cfg.MinSize.Y == 0 || cfg.MinSize.Y < cfg.MinHeight) {
		cfg.MinSize.Y = cfg.MinHeight
	}
	if cfg.MaxWidth > 0 && (cfg.MaxSize.X == 0 || cfg.MaxSize.X > cfg.MaxWidth) {
		cfg.MaxSize.X = cfg.MaxWidth
	}
	if cfg.MaxHeight > 0 && (cfg.MaxSize.Y == 0 || cfg.MaxSize.Y > cfg.MaxHeight) {
		cfg.MaxSize.Y = cfg.MaxHeight
	}
	opts := []app.Option{
		app.Title(cfg.Title),
		app.Size(unit.Dp(cfg.Size.X), unit.Dp(cfg.Size.Y)),
	}
	if cfg.Resizable {
		if cfg.MinSize != (image.Point{}) {
			opts = append(opts, app.MinSize(unit.Dp(cfg.MinSize.X), unit.Dp(cfg.MinSize.Y)))
		}
		if cfg.MaxSize != (image.Point{}) {
			opts = append(opts, app.MaxSize(unit.Dp(cfg.MaxSize.X), unit.Dp(cfg.MaxSize.Y)))
		}
	} else {
		opts = append(opts,
			app.MinSize(unit.Dp(cfg.Size.X), unit.Dp(cfg.Size.Y)),
			app.MaxSize(unit.Dp(cfg.Size.X), unit.Dp(cfg.Size.Y)),
		)
	}
	opts = append(opts, app.Decorated(!cfg.Frameless))
	switch cfg.WindowStartState {
	case WindowStartMinimized:
		opts = append(opts, app.Minimized.Option())
	case WindowStartMaximized:
		opts = append(opts, app.Maximized.Option())
	case WindowStartFullscreen:
		opts = append(opts, app.Fullscreen.Option())
	default:
		// Fall back to the individual flags.
		if cfg.Fullscreen {
			opts = append(opts, app.Fullscreen.Option())
		} else if cfg.Maximized {
			opts = append(opts, app.Maximized.Option())
		} else if cfg.Minimized {
			opts = append(opts, app.Minimized.Option())
		}
	}
	if cfg.AlwaysOnTop {
		opts = append(opts, app.TopMost(true))
	}
	if cfg.Orientation != OrientationAny {
		opts = append(opts, cfg.Orientation.option())
	}
	a := &App{cfg: cfg, themeCheck: make(chan struct{}, 1)}
	a.closeToTray.Store(cfg.CloseToTray)
	a.minimizeToTray.Store(cfg.MinimizeToTray)
	a.topMost.Store(cfg.AlwaysOnTop)
	a.title.Store(&cfg.Title)
	currentApp.Store(a)
	a.platformInit()
	if cfg.Locale != "" {
		i18n.SetLocale(cfg.Locale)
	} else {
		i18n.SetLocale(i18n.SystemLocale())
	}
	if cfg.Intl != nil {
		if err := cfg.Intl.load(); err != nil {
			a.logf(LogLevelError, "giode: translations: %v", err)
		}
	}
	i18n.OnChange(func() {
		// New text and maybe a new reading direction.
		elements.InvalidateCaches()
		a.window.Invalidate()
	})
	if cfg.SingleInstanceLock == nil {
		// Restarted by an update: let the old process finish first.
		host.WaitForUpdate()
	}
	a.setIconBytes(cfg.Icon)
	if runtime.GOOS == "android" || runtime.GOOS == "ios" {
		// Phones run one copy of an app anyway; the system brings it
		// back rather than starting another.
		cfg.SingleInstanceLock = nil
		a.cfg.SingleInstanceLock = nil
	}
	if cfg.SingleInstanceLock != nil {
		a.instance = host.LockInstance(cfg.SingleInstanceLock.UniqueID)
		a.secondInstance = a.instance.Running()
	}
	a.rendererErr = setupRenderer(cfg)
	installFileDrag(a)
	a.window = new(app.Window)
	a.SetMaxFPS(cfg.MaxFPS)
	a.window.Option(opts...)
	a.explorer = explorer.NewExplorer(a.window)
	elements.SetLinkOpener(a.openLink)
	elements.SetLongPressFeedback(func() { a.Haptic(HapticHeavy) })
	if cfg.Center {
		a.window.Perform(system.ActionCenter)
	}
	state.SetInvalidator(a.window.Invalidate)
	if cfg.Logger != nil && cfg.LogLevel <= LogLevelDebug {
		elements.SetDebugLogger(cfg.Logger.Printf)
	}
	properties.SetThemeChangeHook(a.window.Invalidate)
	if cfg.MCP || os.Getenv("GIODE_MCP") == "1" {
		a.mcp = newMCPBridge(a, cfg.MCPOptions)
	}
	return a
}

// --- Window actions --------------------------------------------------------

// SetFullscreen switches the window in or out of fullscreen mode.
func (a *App) SetFullscreen(v bool) {
	if v {
		a.window.Option(app.Fullscreen.Option())
	} else {
		a.window.Option(app.Windowed.Option())
	}
}

// SetTitle changes the window's title, shown in its title bar, the
// taskbar and the window switcher - the playing file's name, say.
// Config.Title is the first. Safe to call from any goroutine.
func (a *App) SetTitle(title string) {
	a.title.Store(&title)
	a.window.Option(app.Title(title))
}

// Title returns the window's title, as set by Config.Title or
// SetTitle.
func (a *App) Title() string {
	if t := a.title.Load(); t != nil {
		return *t
	}
	return a.cfg.Title
}

// Minimize minimizes the window.
func (a *App) Minimize() {
	a.window.Perform(system.ActionMinimize)
}

// Maximize maximizes the window.
func (a *App) Maximize() {
	a.window.Perform(system.ActionMaximize)
}

// Unmaximize restores the window from its maximized state.
func (a *App) Unmaximize() {
	a.window.Perform(system.ActionUnmaximize)
}

// SetSize resizes the window to w x h dp, leaving fullscreen or
// maximized state first. The size bounds from Config (MinSize,
// MaxSize, MinWidth, ...) are widened when they'd prevent it, so
// automation can shrink a desktop app to a phone's size; a window
// that isn't Resizable stays locked, at the new size.
func (a *App) SetSize(w, h int) {
	opts := []app.Option{app.Windowed.Option(), app.Size(unit.Dp(w), unit.Dp(h))}
	if !a.cfg.Resizable {
		a.cfg.MinSize, a.cfg.MaxSize = image.Pt(w, h), image.Pt(w, h)
	} else {
		if a.cfg.MinSize.X > w || a.cfg.MinSize.Y > h {
			a.cfg.MinSize = image.Pt(min(a.cfg.MinSize.X, w), min(a.cfg.MinSize.Y, h))
		}
		if a.cfg.MaxSize != (image.Point{}) && (a.cfg.MaxSize.X < w || a.cfg.MaxSize.Y < h) {
			a.cfg.MaxSize = image.Pt(max(a.cfg.MaxSize.X, w), max(a.cfg.MaxSize.Y, h))
		}
	}
	if a.cfg.MinSize != (image.Point{}) {
		opts = append(opts, app.MinSize(unit.Dp(a.cfg.MinSize.X), unit.Dp(a.cfg.MinSize.Y)))
	}
	if a.cfg.MaxSize != (image.Point{}) {
		opts = append(opts, app.MaxSize(unit.Dp(a.cfg.MaxSize.X), unit.Dp(a.cfg.MaxSize.Y)))
	}
	a.window.Option(opts...)
}

// Center centers the window on the screen.
func (a *App) Center() {
	a.window.Perform(system.ActionCenter)
}

// Raise brings the window to the front, restoring it first when it's
// minimized. On Windows it may be called from any goroutine; elsewhere
// call it on the UI goroutine (from a handler, or through Post).
//
// Windows lets an app take the foreground only when the user just
// interacted with it, or another app passed it the right (a second
// launch does, see SingleInstanceLock); otherwise the taskbar button
// flashes instead.
func (a *App) Raise() {
	if a.raiseNative() {
		return
	}
	a.window.Perform(system.ActionRaise)
}

// Close requests the window to close, ending the app - also with
// CloseToTray, which only applies to the user closing it.
//
// It may be called from any goroutine: the close itself runs on the UI
// goroutine (Gio's Perform isn't safe elsewhere), which Post reaches
// even while the window is hidden.
func (a *App) Close() {
	a.quitting.Store(true)
	a.Post(func() { a.window.Perform(system.ActionClose) })
}

// RequestClose closes the window as the system's close button does:
// with CloseToTray it hides the window to the tray, otherwise it ends
// the app, as Close does. TitleBar's close button calls it. It may be
// called from any goroutine.
func (a *App) RequestClose() {
	if a.closesToTray() {
		a.HideWindow()
		return
	}
	a.Close()
}

// Window exposes the underlying *app.Window for low-level
// customization.
func (a *App) Window() *app.Window {
	return a.window
}
