package giode

import (
	"image"
	"net/url"

	"gioui.org/io/event"
	"gioui.org/io/key"
)

// Config configures the application window. The options cover the
// common window settings of desktop applications; unsupported
// platform features are simply ignored.
type Config struct {
	// Title is the window title.
	Title string
	// Size is the initial window size. Defaults to 800x600.
	Size image.Point
	// MinSize and MaxSize bound the window size when Resizable. They
	// combine with MinWidth/MinHeight/MaxWidth/MaxHeight below; the
	// tighter bound wins.
	MinSize image.Point
	MaxSize image.Point
	// MinWidth, MinHeight, MaxWidth and MaxHeight bound the window
	// size per axis.
	MinWidth  int
	MinHeight int
	MaxWidth  int
	MaxHeight int
	// Resizable controls whether the user can resize the window. When
	// false the size is locked to Size.
	Resizable bool
	// Frameless removes the OS window frame (title bar and borders).
	Frameless bool
	// Draggable makes the window movable by dragging empty
	// (non-widget) space anywhere in it. It only applies to frameless
	// windows. To drag by a custom title bar only, leave it off and
	// use TitleBar, which marks its own area as the drag handle.
	Draggable bool
	// Background is the window background color. Theme tokens
	// ("theme.background") follow SetMode and SetPalette. On Windows,
	// built with the giode CLI's Gio patches, a new window shows it
	// (as it is when New runs) until the first frame, rather than
	// white.
	Background string
	// StatusBar and NavigationBar color Android's status bar and
	// navigation bar. Both default to Background and follow theme
	// changes, so the bars blend with a light or dark app. Other
	// platforms ignore them.
	StatusBar     string
	NavigationBar string
	// Orientation restricts rotation on mobile platforms. Defaults to
	// OrientationAny. Desktop platforms ignore it.
	Orientation Orientation
	// FollowSystemTheme sets the theme mode (SetMode) from the
	// platform's light or dark appearance at startup and whenever it
	// changes. See SystemDarkMode for the platforms that report it.
	FollowSystemTheme bool
	// OnSystemThemeChange runs on the UI goroutine when the platform's
	// appearance changes between light and dark, with or without
	// FollowSystemTheme.
	OnSystemThemeChange func(dark bool)
	// EdgeToEdge lays the view out over the whole window, under the
	// status bar, navigation bar and cutouts, instead of inside the
	// safe area. Pad content with SafeArea where it must stay clear of
	// them. Desktop windows have no insets, so it changes nothing there.
	EdgeToEdge bool
	// Font is the font family of text that doesn't set a FontFamily,
	// Giode's own components included (title bar, buttons, text
	// fields, menus, toasts): register its files with RegisterFont
	// before New. Empty keeps the Go fonts. See SetDefaultFont.
	Font string
	// Icon is the window icon, PNG or JPEG bytes. It is applied at
	// runtime where the platform supports it (Windows). The icon
	// shown by file managers comes from the executable and is set at
	// build time.
	Icon []byte
	// WindowStartState selects the initial window state. It takes
	// precedence over the Fullscreen, Maximized and Minimized flags.
	WindowStartState WindowStartState
	// Fullscreen starts the window in fullscreen mode.
	Fullscreen bool
	// Maximized starts the window maximized.
	Maximized bool
	// Minimized starts the window minimized.
	Minimized bool
	// Center starts the window centered on the screen.
	Center bool
	// AlwaysOnTop keeps the window above other windows.
	AlwaysOnTop bool
	// Logger receives Giode diagnostics at or above LogLevel.
	Logger Logger
	// LogLevel filters Logger output. Defaults to LogLevelDebug.
	LogLevel LogLevel
	// OnStartup runs before the event loop starts.
	OnStartup func()
	// OnReady runs after the first frame is presented to the screen.
	OnReady func()
	// OnShutdown runs after the window closes. On Android and iOS the
	// system may end a backgrounded app without warning, so save state
	// in OnBackground instead.
	OnShutdown func()
	// OnBackground runs when the app leaves the screen: on Android when
	// its activity stops (the user switched apps, went home, or opened
	// a system picker), on desktops when the window is minimized. On
	// Android it's the last reliable moment to save: the system may
	// freeze or end the app afterwards without warning. It runs
	// synchronously on the platform's main thread (Android) or the UI
	// goroutine (desktop): keep it short, and don't wait on the UI
	// (RunOnUI) from it. On iOS it runs on UIApplication's
	// background notification.
	OnBackground func()
	// OnForeground runs when the app comes back after OnBackground,
	// on the same goroutine.
	OnForeground func()
	// OnEvent observes every window event before giode handles it, on
	// the UI goroutine - for Gio extensions that need to see them. It
	// must not call FrameEvent.Frame. OpenFile and SaveFile already
	// get the events they need.
	OnEvent func(event.Event)
	// OnBeforeClose runs when a window close is requested. It cannot
	// veto the close.
	OnBeforeClose func()
	// OnKey receives the keys nothing in the UI took - app-wide
	// shortcuts - after the focused field, dialogs, menus and the
	// Escape and Back handlers have had theirs; see giode.OnKey. It
	// runs on the UI goroutine and returns whether it used the key.
	OnKey func(e key.Event) bool
	// OnWindowStateChange runs when the window is minimized,
	// maximized, made fullscreen or restored - by the app or by the
	// user.
	OnWindowStateChange func(WindowState)
	// Locale is the app's locale ("fa", "pt-BR") for the i18n package
	// and the layout's reading direction; the system's by default. Change
	// it later with SetLocale.
	Locale string
	// Intl, when set, turns on translations: giode loads the message
	// files (<locale>.json) from a folder at startup, and Intl(key,
	// data) reads them. Nil leaves translation off.
	Intl *IntlOptions
	// OnOpenURL runs on the UI goroutine with a link that opened the
	// app or reached it while running: a URL scheme it registered
	// (giode.toml's app.schemes) or one of its universal links / Android
	// App Links (app.links). With a SingleInstanceLock, a link that
	// starts a second desktop instance reaches the running one here.
	OnOpenURL func(u *url.URL)
	// OnShareReceived runs on the UI goroutine with what another app
	// shared with this one from the system share sheet - files, text or
	// both - whether the share started the app or reached it while
	// running. The app must be a share target: giode.toml's
	// app.share_target, which registers it on Android and builds iOS's
	// share extension. The receiver closes the files.
	OnShareReceived func(SharedContent)
	// ContextActions add entries to the file manager's right-click menu
	// that open the app with the chosen paths; see ContextAction. The
	// giode CLI adds the ones in giode.toml too.
	ContextActions []ContextAction
	// OnContextAction runs on the UI goroutine when the app was started
	// from one of its context actions, with the action's ID and the
	// paths it was chosen on. With a SingleInstanceLock, a context
	// action chosen while the app runs reaches the running instance
	// here.
	OnContextAction func(id string, paths []string)
	// MinimizeToTray hides the window when it's minimized, so it leaves
	// the taskbar or Dock and lives in the tray until ShowWindow (by
	// default a click on the tray icon) brings it back. It needs a tray
	// icon: set one up with App.Tray. Desktop only. App.SetMinimizeToTray
	// changes it while the app runs.
	MinimizeToTray bool
	// CloseToTray hides the window when the user closes it, instead of
	// ending the app; Quit (a tray menu item, say) ends it. Windows and
	// macOS; on Linux, where Gio can't keep a closed window, closing
	// still quits. App.SetCloseToTray changes it while the app runs.
	CloseToTray bool
	// StartInTray starts the app with its window hidden, in the tray:
	// StartInTrayAtLogin when the system started it at login (see
	// App.SetLaunchAtLogin), StartInTrayAlways every time. The window
	// shows with ShowWindow, from the tray icon. It needs a tray icon
	// (App.Tray) and is ignored where TraySupported is false.
	StartInTray StartInTray
	// MenuBarApp makes a macOS app live in the menu bar: no Dock icon
	// and no app menu, as menu bar utilities do. Pair it with App.Tray.
	// Other platforms ignore it.
	MenuBarApp bool
	// SingleInstanceLock, when set, ensures only one instance of the
	// application runs on desktops. Android and iOS ignore it: they run
	// one copy of an app anyway. When the lock can't be written (no
	// usable temporary directory), the app runs as if it were alone.
	SingleInstanceLock *SingleInstanceLock
	// OnFileDrop runs with the paths of files dropped onto the window
	// from the file manager, on the UI goroutine. It works on Windows
	// (when the app runs as administrator too), macOS, and on Linux on
	// X11 and Wayland when an up to date giode CLI builds the app: its
	// Gio patches receive the drops. Elsewhere a warning is logged once.
	OnFileDrop func(paths []string)
	// OnFileDrag runs on the UI goroutine as files are dragged over
	// the window - entering, moving, leaving, dropped - for
	// highlighting where they can go; App.FileDrag reports the same
	// during layout. Setting it accepts drops as OnFileDrop does. On
	// Windows, an app running as administrator gets only the drop.
	OnFileDrag func(FileDrag)
	// MCP starts a Model Context Protocol server alongside the window,
	// so AI agents can see and drive the running app: inspect the UI,
	// click, type, take screenshots, read and set named state - plus
	// any tools added with App.MCPTool. It listens on
	// http://127.0.0.1:7600/mcp unless MCPOptions say otherwise.
	// Setting the GIODE_MCP=1 environment variable enables it too,
	// without rebuilding.
	MCP bool
	// MCPOptions configures the MCP server.
	MCPOptions MCPOptions
	// ShowFrameTimes draws a small counter in the top right corner with
	// how long frames take to build and to render, and the frame rate:
	// a development aid for finding what makes scrolling or a stream
	// stutter on a device. App.FrameStats returns the same numbers, and
	// the MCP server reports them as frame_stats.
	ShowFrameTimes bool
	// MaxFPS caps the frame rate: frames are drawn at most this many
	// times a second, saving CPU, GPU and battery during animations,
	// scrolling and video. 0 (the default) draws at the display's
	// refresh rate. Frames still follow the display's refresh, so on a
	// 60 Hz display a cap of 30 gives 30 fps but one of 50 gives 30 too:
	// each frame waits for the refresh after its interval. Change it
	// while the app runs with SetMaxFPS.
	MaxFPS int

	// Underlay draws caller-rendered GPU content under the UI each
	// frame - video frames from libmpv, a 3D or map engine - without
	// copying it off the GPU. It's called on the window's render thread
	// with the window's GPU context current, before the UI is drawn
	// over its output; Background isn't painted, so the UI's own
	// backgrounds cover only what they should. Call App.Invalidate from
	// any goroutine when a new frame is ready. Engines drawing with
	// OpenGL set Renderer to RendererOpenGL. It needs giode's patched
	// Gio (giode build and giode run); without it Run returns an error.
	Underlay func(target GPUTarget)
	// UnderlayRelease runs on the render thread with the GPU context
	// still current, just before Gio releases its GPU resources: when
	// the window closes or its view goes away (Android leaving the
	// screen), or the context is lost and made again, after which
	// Underlay runs in the new one. Free the engine's GPU resources
	// here (for libmpv, mpv_render_context_free). It needs an up to
	// date giode CLI (the giode_gio2 patches); with older ones Run
	// returns an error.
	UnderlayRelease func()
	// Renderer selects the window's GPU API; see RendererOpenGL.
	Renderer Renderer
}

// Logger receives Giode diagnostics. Any *log.Logger satisfies it.
type Logger interface {
	Printf(format string, v ...any)
}

// SingleInstanceLock prevents a second instance of the application
// from running. When a second instance launches, its
// OnSecondInstanceLaunch callback runs and the instance exits without
// showing a window.
type SingleInstanceLock struct {
	// UniqueID identifies the application; it must be unique per app.
	UniqueID string
	// OnSecondInstanceLaunch runs in the second instance in place of
	// the UI.
	OnSecondInstanceLaunch func()
	// OnLaunchArgs runs in the running instance, on the UI goroutine,
	// with the command-line arguments (without the program name) of a
	// second instance that started and exited - a file opened with the
	// app, a URL. The window comes to the front either way. Context
	// actions go to Config.OnContextAction instead.
	OnLaunchArgs func(args []string)
}

// WindowStartState selects the initial state of the window.
type WindowStartState int

const (
	// WindowStartNormal starts the window in its normal state.
	WindowStartNormal WindowStartState = iota
	// WindowStartMinimized starts the window minimized.
	WindowStartMinimized
	// WindowStartMaximized starts the window maximized.
	WindowStartMaximized
	// WindowStartFullscreen starts the window fullscreen.
	WindowStartFullscreen
)

// LogLevel filters the messages passed to Logger.
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarning
	LogLevelError
)
