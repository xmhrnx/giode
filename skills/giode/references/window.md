# Window configuration

`giode.New(giode.Config{...})` creates the app. The options cover the common window
settings of desktop applications; unsupported platform features are ignored.

## Config fields

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `Title` | `string` | `""` | Window title. |
| `Size` | `image.Point` | 800x600 | Initial window size in dp. |
| `MinSize` / `MaxSize` | `image.Point` | — | Size bounds when resizable. |
| `MinWidth` / `MinHeight` / `MaxWidth` / `MaxHeight` | `int` | — | Per-axis size bounds; combine with `MinSize`/`MaxSize` (tighter bound wins). |
| `Resizable` | `bool` | `false` | Allow user resizing. When false the size is locked to `Size`. |
| `Frameless` | `bool` | `false` | Remove the OS title bar and borders. |
| `Draggable` | `bool` | `false` | Drag the window from empty (non-widget) space anywhere. Frameless only. For a custom title bar, use [TitleBar](components/titlebar.md) instead. |
| `OnFileDrop` | `func(paths []string)` | `nil` | Files dropped onto the window from the file manager, on the UI goroutine; see [File drop](#file-drop). |
| `OnFileDrag` | `func(FileDrag)` | `nil` | Files dragged over the window: enter, over, leave, drop, with the position. |
| `OnWindowStateChange` | `func(WindowState)` | `nil` | Runs when the window is minimized, maximized, made fullscreen or restored. |
| `Background` | `string` | `"#000000"` | Window background color (any CSS hex, or a theme token such as `"theme.background"`, which follows `SetMode`/`SetPalette`). |
| `StatusBar` / `NavigationBar` | `string` | `Background` | Colors of Android's status and navigation bars. They follow theme changes, so the bars match a light or dark app. Ignored elsewhere. |
| `Orientation` | `Orientation` | `OrientationAny` | Rotation on mobile: `OrientationAny`, `OrientationPortrait`, `OrientationLandscape`. Ignored on desktop. |
| `Icon` | `[]byte` | `nil` | Window icon, PNG or JPEG bytes. Applied at runtime on Windows. |
| `WindowStartState` | `WindowStartState` | `WindowStartNormal` | Initial state: `WindowStartMinimized`, `WindowStartMaximized`, `WindowStartFullscreen`. Takes precedence over the flags below. |
| `Fullscreen` / `Maximized` / `Minimized` | `bool` | `false` | Start in the given state (used when `WindowStartState` is `WindowStartNormal`). |
| `Center` | `bool` | `false` | Start centered on the screen. |
| `AlwaysOnTop` | `bool` | `false` | Keep above other windows; `SetAlwaysOnTop` changes it later. |
| `Logger` | `Logger` | `nil` | Receives diagnostics; any `*log.Logger` satisfies it. |
| `LogLevel` | `LogLevel` | `LogLevelDebug` | `LogLevelDebug`, `LogLevelInfo`, `LogLevelWarning`, `LogLevelError`. |
| `MaxFPS` | `int` | `0` | Cap the frame rate at this many frames a second, saving CPU, GPU and battery during animations and scrolling; 0 draws at the display's refresh rate. `SetMaxFPS` changes it while the app runs. Frames still follow the display's refresh: on a 60 Hz display a cap of 30 gives 30 fps, one of 50 also gives 30. |
| `ShowFrameTimes` | `bool` | `false` | Draw a counter with frame build and render times and the frame rate, for finding stutter on a device; see [performance.md](performance.md#measuring-frame-times). |
| `OnStartup` | `func()` | `nil` | Runs before the event loop starts. |
| `OnReady` | `func()` | `nil` | Runs after the first frame is presented. |
| `OnShutdown` | `func()` | `nil` | Runs after the window closes. Mobile systems may end a backgrounded app without it running: save in `OnBackground`. |
| `OnBackground` / `OnForeground` | `func()` | `nil` | The app left the screen / came back; see [Background and foreground](#background-and-foreground). |
| `OnEvent` | `func(event.Event)` | `nil` | Observes every window event before giode handles it, for Gio extensions that need them. Must not call `FrameEvent.Frame`. |
| `OnBeforeClose` | `func()` | `nil` | Runs when a close is requested. It cannot veto the close. |
| `SingleInstanceLock` | `*SingleInstanceLock` | `nil` | Ensures only one instance runs; see below. |

## Single instance

```go
app := giode.New(giode.Config{
	SingleInstanceLock: &giode.SingleInstanceLock{
		UniqueID: "c9c8fd93-6758-4144-87d1-34bdb0a8bd60",
		OnSecondInstanceLaunch: func() {
			fmt.Println("already running")
		},
	},
})
```

When a second instance launches, its `OnSecondInstanceLaunch` callback runs and the
instance exits without showing a window. The lock survives crashes (a dead owner's lock
is reclaimed). Android and iOS ignore the lock (the system runs one copy of an app), and
a lock file that can't be created counts as no other copy running, so the app starts.

## Runtime window actions

```go
app.SetFullscreen(true) // or false to leave fullscreen
app.Minimize()
app.Maximize()
app.Unmaximize()
app.Center()
app.Raise()
app.Close()
app.SetSize(400, 840) // in dp; widens Min/Max bounds when needed
app.SetAlwaysOnTop(true) // app.AlwaysOnTop() reports it
app.SetTitle("Playing: movie.mkv") // Config.Title is the first; app.Title() reads it
```

`app.Window()` returns the underlying `*app.Window` for anything not covered here.

## Position and screen

```go
s, err := app.Screen()             // the monitor showing the window, in dp
// s.Bounds, s.WorkArea (without the taskbar, Dock, menu bar or panels), s.Scale
f, err := app.Frame()              // the window's area, title bar and borders included
app.SetPosition(40, 40)            // top left corner, in dp from the primary screen's top left
app.MoveTo(giode.CornerBottomRight, 16) // a mini player in a corner of the work area
```

| Platform | Position and screen |
| --- | --- |
| Windows | Yes, per monitor; the visible frame, without Windows 10's invisible borders. |
| macOS | Yes; in points, which are dp. |
| Linux, X11 | Yes; the monitor from XRandR (loaded at run time), the work area from `_NET_WORKAREA`. |
| Linux, Wayland | `ErrNotSupported`: compositors place windows and don't tell apps where. |
| Android, iOS, web | `Screen` and `Frame` are the window, the work area without the system bars; `SetPosition` returns `ErrNotSupported`. |

`SetAlwaysOnTop` works on Windows, macOS and X11; Wayland leaves it to the user. All of
these are safe from any goroutine.

## Version

`giode.AppVersion` and `giode.AppBuild` hold `version` and `build` from `giode.toml`: the
giode CLI sets them when it builds or runs a desktop app. A plain `go build` leaves them
empty.

## Draggable windows

`Draggable: true` registers the window background as a native move area only while the
pointer hovers empty space, so widgets keep receiving clicks. The drag is OS-native: the
window moves with the mouse like a title-bar drag.

## Window icon

Pass PNG or JPEG bytes and the icon is applied to the window (title bar and taskbar) at
runtime on Windows:

```go
icon, _ := os.ReadFile("app.png") // or embed the bytes with go:embed
app := giode.New(giode.Config{Icon: icon})
```

The icon shown by file managers comes from the executable, not the running window, and
is embedded at build time instead:

```sh
rsrc -ico app.ico -o app.syso   # or goversioninfo / windres
go build
```

## Opening and saving files

```go
go func() {
	f, err := app.OpenFile(ctx, giode.FileFilter{Extensions: []string{"wav", "mp3"}})
	if errors.Is(err, giode.ErrCanceled) {
		return
	}
	if err != nil { ... }
	defer f.Close()
	data, err := io.ReadAll(f)
	app.Post(func() { load(data) }) // back to the UI goroutine
}()
```

`OpenFile`, `OpenFiles` and `SaveFile(ctx, "speech.wav")` show the platform's own picker:
the system picker on Android and iOS, the only way apps there can reach the user's files,
and the native dialogs on desktops: IFileOpenDialog on Windows and NSOpenPanel on macOS
(both attached to the app's window, choosing several files with `OpenFiles`), the XDG
desktop portal on Linux. The rest is built on `gioui.org/x/explorer`, which giode feeds
the window events it needs.

They block until the user decides, so **call them from a goroutine**, never from a click
handler: the picker needs the event loop running. `ErrCanceled` means the user closed the
picker; when `ctx` ends first they return `ctx.Err()` (the picker may stay open, and a file
chosen after that is closed). One dialog shows at a time.

For other Gio extensions that need the window's events, set `Config.OnEvent`.

## Sharing

```go
err := app.Share(ctx, giode.ShareFile{Path: p, MIME: "audio/wav"})
err := app.Share(ctx, giode.ShareText{Text: link, Subject: "Listen to this"})
```

On Android, `Share` opens the system share sheet. A shared file is first copied into the
device's Downloads folder: apps built with gogio can't declare the `FileProvider` Android
otherwise requires for handing files to other apps, and MediaStore is the provider left.
This needs Android 10 or later. The helper is a small Java class shipped as a `.jar` in
`internal/mobile`, which gogio packs into the APK automatically.

On desktops, `ShareText` copies the text to the clipboard and `ShareFile` shows the file in
the file manager (Explorer, Finder, or the freedesktop file manager on Linux). On iOS it opens
the share sheet (`UIActivityViewController`, a popover on iPad) with the file itself; on
the web `Share` returns `ErrNotSupported`. A large file takes a moment to copy, so
call `Share` from a goroutine too.

## Background and foreground

`OnBackground` runs when the app leaves the screen, `OnForeground` when it comes back:

- **Android**: when the activity stops and starts again - the user switched apps, went
  home, locked the phone, or opened a system picker. Android may freeze the app soon after
  and end it later without warning, so **`OnBackground` is the last reliable moment to
  save** (a draft, unsent input). Don't rely on `OnShutdown` or `OnBeforeClose` there.
- **Desktop**: when the window is minimized and restored (on Windows with the giode CLI's
  Gio patches, which make Gio report minimized windows).
- **iOS**: when the app enters the background and returns (`UIApplication`
  notifications). iOS suspends a backgrounded app within seconds, so save here too;
  `KeepAlive` buys about 30 seconds (see [platform.md](platform.md#staying-awake)).

On Android they run synchronously on the platform's main thread, on desktops on the UI
goroutine. Keep them short, make any state they touch safe to use from another goroutine
(a `State`, or a lock), and don't wait on the UI (`RunOnUI`) from them.

## System tray and the macOS menu bar

`app.Tray()` puts an icon in the system tray: the notification area on Windows, the menu
bar on macOS, the status notifier area on Linux (KDE and most desktops; GNOME needs the
AppIndicator extension).

```go
a := giode.New(giode.Config{
	Title:          "Pocket Agent",
	Icon:           iconPNG,
	MinimizeToTray: true, // minimizing hides the window to the tray
	CloseToTray:    true, // so does closing it; Quit ends the app
})
a.Tray().
	SetTemplateIcon(trayPNG). // tinted to match the macOS menu bar
	SetTooltip("Pocket Agent").
	SetMenu(
		giode.TrayAction("Open", a.ShowWindow),
		giode.TrayCheckbox("Do not disturb", dnd, toggleDND),
		giode.TraySubmenu("Recent", recentItems...),
		giode.TraySeparator(),
		giode.TrayAction("Quit", a.Quit),
	)
```

| Method | Effect |
| --- | --- |
| `SetIcon(png)` / `SetTemplateIcon(png)` | The icon; without one, `Config.Icon`. A template icon is black on transparency and macOS tints it for light and dark menu bars. |
| `SetTitle(text)` | Text next to the icon in the **macOS menu bar** (and Linux panels that show it): an unread count, a timer, a status. Update it whenever you like. |
| `SetTooltip(text)` | Hover text. |
| `SetMenu(items...)` | Replaces the menu. Call it again to change items or check marks. Item callbacks run on the UI goroutine. |
| `OnClick(fn)` | What a click on the icon does. By default it shows or hides the window on Windows and Linux; on macOS a click opens the menu. |
| `Remove()` | Takes the icon away. |

The window can hide without closing: `HideWindow()`, `ShowWindow()` and `WindowVisible()`.
A hidden window leaves the screen and the taskbar or Dock, and comes back as it was;
clicking the Dock icon brings it back on macOS. `Config.MenuBarApp` makes a macOS app a
menu bar utility, with no Dock icon or app menu. `Quit()` ends the app even with
`CloseToTray`, as does `Close()`. `RequestClose()` closes the way the system close
button does: it hides the window with `CloseToTray`, and closes it otherwise. A `TitleBar`'s
close button calls it.

Tray menu items and `OnClick` run on the UI goroutine even while the window is hidden and
draws no frames, as does anything queued with `Post`. The default tray click and a second
launch (with `SingleInstanceLock`) show the window directly, so a hidden window always
comes back; `HideWindow`, `ShowWindow` and `Quit` may be called from any goroutine. So may
the tray's setters (`SetMenu`, `SetTitle`, `SetIcon`...), from a menu item's callback too:
they never wait for the tray, which shows the change a moment later.

Platform notes: on Linux, hiding works under X11. Wayland can't hide a window, nor bring a
minimized one back, so there `HideWindow` and `StartInTray` leave the window on screen, and
`CloseToTray` isn't available (closing quits). Android, iOS and the
web have no tray: `TraySupported()` reports false and the calls do nothing.

`SetCloseToTray(on)` and `SetMinimizeToTray(on)` change those options while the app runs,
from a settings screen say, and `CloseToTray()` and `MinimizeToTray()` read them back.
They take effect straight away; no restart.

## Start at login

`app.SetLaunchAtLogin(true)` registers the app to start when the user signs in, and
`SetLaunchAtLogin(false)` removes the registration. `app.LaunchAtLogin()` reports the
current state: read it rather than remembering it, since the user can turn it off in the
system's settings.

```go
a := giode.New(giode.Config{
	Title:       "Pocket Agent",
	CloseToTray: true,
	StartInTray: giode.StartInTrayAtLogin, // started at login: no window, just the tray icon
})

// In Settings:
login := giode.Input.Switch("Start at login").OnChange(func(on bool) {
	if err := a.SetLaunchAtLogin(on); err != nil {
		log.Printf("start at login: %v", err)
	}
})
login.SetChecked(a.LaunchAtLogin())
```

`giode.LaunchedAtLogin()` reports whether this launch came from the registration, to start
quietly: skip a welcome screen, don't steal focus. `Config.StartInTray` hides the window
as soon as it exists: `StartInTrayAtLogin` only when started at login, `StartInTrayAlways`
every time. `ShowWindow()` (a click on the tray icon by default) brings it up. Set up a
tray icon with it, or the user has no way to open the window.

| Platform | Registration |
| --- | --- |
| Windows | A value under `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`. Task Manager's Startup apps page can turn it off; `LaunchAtLogin` then reports false, and `SetLaunchAtLogin(true)` turns it back on. |
| Windows, MSIX | The package's startup task. Set `startup_task = true` under `[windows.msix]` in giode.toml so the CLI declares it in the manifest. If the user turned it off in Settings > Apps > Startup, only they can turn it back on: `SetLaunchAtLogin(true)` returns an error saying so. |
| macOS | A LaunchAgent, `~/Library/LaunchAgents/<id>.login.plist`, that opens the app bundle. |
| Linux | An XDG autostart entry, `~/.config/autostart/<id>.desktop`. It runs the AppImage or the Flatpak (`flatpak run <id>`) when the app is one. A Flatpak needs `--filesystem=xdg-config/autostart:create` to write it. |
| Android, iOS, web | `SetLaunchAtLogin` returns `ErrNotSupported`. |

The entry's file is named after `app.id` from giode.toml, and the startup apps list shows
`app.name` with the app's icon: the CLI builds both into the app (`giode.AppID`,
`giode.AppName`). Built without the CLI, set them yourself, or the window title is used. On
Linux the icon is the Flatpak's, else the one a deb or rpm package installed under the app
ID, else `Config.Icon` (saved to `~/.local/share/icons`). On macOS the LaunchAgent names the
app's bundle ID, so Login Items lists it under the app.

## File manager context menu

Context actions add entries to the file manager's right-click menu that open the app with
the chosen files or folders - "Ask Pocket Agent about this file", "Open Pocket Agent here".
Declare them in `giode.toml` (see the CLI's README) or in `Config.ContextActions`, and
handle them in `Config.OnContextAction`:

```go
a := giode.New(giode.Config{
	SingleInstanceLock: &giode.SingleInstanceLock{UniqueID: "com.example.agent"},
	ContextActions: []giode.ContextAction{
		{ID: "ask", Label: "Ask Pocket Agent", Extensions: []string{".md", ".txt"}},
		{ID: "here", Label: "Open Pocket Agent here", Folders: true, FolderBackground: true},
	},
	OnContextAction: func(id string, paths []string) {
		switch id {
		case "ask":
			chat.Attach(paths...)
		case "here":
			chat.SetWorkingDir(paths[0])
		}
		a.ShowWindow()
	},
	OnStartup: func() { a.InstallContextActions() },
})
```

| Platform | Where they appear | How they're registered |
| --- | --- | --- |
| Windows | Explorer's context menu (on Windows 11, under "Show more options") | The setup program from `giode build windows-installer`, or `App.InstallContextActions()` for the current user (no admin rights) |
| macOS | Finder's Quick Actions / Services submenu | The `NSServices` entries `giode build macos` writes; macOS picks them up once the app is in `/Applications` and has run once |
| Linux | GNOME Files and Caja (under Scripts), Dolphin, Nemo | `App.InstallContextActions()`, into the user's home |

Choosing an action starts the app with `--giode-action=<id> <paths>`; giode parses that and
runs `OnContextAction` on the UI goroutine. With a `SingleInstanceLock`, an action chosen
while the app runs goes to the running instance instead of starting a second one.
`UninstallContextActions()` removes what `InstallContextActions` added.

On Windows, Explorer would start the app once per selected file for a plain menu command,
so actions on files and folders are registered as drop targets instead: Explorer hands the
whole selection to the running app in one call (one `OnContextAction` with every path), and
starts the app only when it isn't running. The app's window comes to the front and is
restored if it was minimized; the launching copy passes on its right to take the
foreground, so Windows doesn't just flash the taskbar button. This needs a 64-bit build;
32-bit builds fall back to one launch per file, which the single-instance lock gathers
into the running copy.

The single-instance lock now also passes any second launch's arguments to the running
instance: `SingleInstanceLock.OnLaunchArgs` receives them (a file opened with the app,
say), and the window comes to the front.

## Not supported

A few common window features are deliberately absent because Gio does not expose them:
closable/minimizable/maximizable flags, window transparency,
native application menus and power-state callbacks (suspend/resume).

## Clipboard

```go
app.SetClipboard("copied text")             // from any goroutine; applied on the next frame
app.ReadClipboard(func(text string) { ... }) // fn runs on the UI goroutine
```

Inside a `Layout`, `kit.WriteClipboard(gtx, text)` copies immediately. For a ready-made
control, `giode.CopyButton(func() string { return value })` copies when clicked and shows a
check mark for a moment (`Labels`, `Feedback`, `OnCopy` customize it).

## Back button

`Dialog`, `Drawer` and `Menu` close on Escape and on the system Back button (Android)
while open, and `Router` goes back while it has history. For anything else, register a
handler from `Layout`, every frame, for as long as Back has something to do:

```go
func (p *EditorPage) Layout(gtx giode.Context) giode.Dimensions {
	if p.dirty {
		giode.HandleBack(func() { p.confirmDiscard = true })
	}
	...
}
```

The handler registered last in the frame wins, so an overlay on top takes Back before the
page beneath it, and only one handler runs. Handlers run after the frame's layout, so they
may change anything the view reads. When nothing is registered, Back is left to the
platform: on Android it closes the app as usual. `HandleBack` reacts to Back only;
`elements.HandleDismiss` also reacts to Escape and is what overlays use.

Handlers are registered during layout rather than consulted when Back is pressed because
Android decides whether the app handled Back the moment it's pressed, from the handlers
registered in the last frame - a callback that returned false later could no longer hand
the press back to the system.

## Keyboard shortcuts

Register app-wide shortcuts with `giode.Shortcuts`, from the view every frame:

```go
giode.Shortcuts(
	giode.Shortcut("Mod+T", newTerminalTab), // Mod is Command on macOS, Ctrl elsewhere
	giode.Shortcut("Mod+K", focusSearch),
	giode.Shortcut("Mod+,", openSettings),
	giode.Shortcut("Mod+Shift+W", closeWindow),
	giode.Shortcut("F5", refresh),
)
```

A spec is modifiers and a key joined by `+`. The modifiers are `Mod`, `Ctrl`, `Shift`,
`Alt` (or `Option`) and `Super` (or `Cmd`, `Win`). The key is a letter, a digit,
punctuation, `F1`–`F12`, `Enter`, `Esc`, `Tab`, `Space`, `Backspace`, `Delete`, `Home`,
`End`, `PageUp`, `PageDown` or an arrow (`Up`, `Down`, `Left`, `Right`). A shortcut matches
its exact modifiers. `giode.FormatShortcut("Mod+Shift+T")` shows it the platform's way
(`⇧⌘T`, `Ctrl+Shift+T`). `Menu.Shortcut("Mod+O")` does the same for a menu item's label.
The `shortcut` package parses specs entered by the user (`shortcut.Parse`).

For anything else, `Config.OnKey`, or `giode.OnKey(fn)` called from the view every frame
(the way `HandleBack` is), gets the raw key events:

```go
app := giode.New(giode.Config{
	OnKey: func(e key.Event) bool {
		if e.State != key.Press {
			return false
		}
		switch {
		case e.Name == "O" && e.Modifiers.Contain(key.ModShortcut):
			openFile()
		case e.Name == key.NameSpace:
			togglePlay()
		default:
			return false
		}
		return true
	},
})
```

Shortcuts and `OnKey` handlers only get the keys nothing else took. The focused text field, dialogs, menus
and other portals, and the Escape and Back handlers (`HandleDismiss`, `HandleBack`) all come
first, whatever order the view lays things out in. While a text field has focus, the keys
that type into it (a character or Space, alone or with Shift) never reach a shortcut, so
typing "h" or a space into a search box doesn't run an `H` or `Space` shortcut. Shortcuts
with `Mod`, `Ctrl` or `Alt`, function keys and Escape still fire, and Delete, Backspace and
the arrows go to the field. Escape closes an open dialog before it reaches the shortcut. A
custom text widget gets the same protection by calling `elements.MarkTyping()` every
frame it has focus. The handler gets
presses and releases (check `e.State`) on the UI goroutine. It returns whether it used the
key. A custom host gets the same behavior from `giode.Frame`.

Tab and Shift+Tab move the keyboard focus unless something asks for them by name: the
window hands them only to a listener that names the key. A `Shortcut("Tab", …)` or
`Shortcut("Shift+Tab", …)` does that, so it fires instead of moving the focus; Tab with
other modifiers (Ctrl+Tab) still moves it unless a shortcut names that too. A handler of
your own names them as it registers:

```go
giode.OnKey(complete, giode.NamedKey{Name: key.NameTab})
```

On phones, the arrow keys move the focus the same way, and a shortcut naming `Up`,
`Down`, `Left` or `Right` takes them.

## File drop

```go
giode.Config{
	OnFileDrop: func(paths []string) {
		model.files = append(model.files, paths...) // UI goroutine: no locks needed
	},
}
```

To highlight where files can go while they're dragged, use `OnFileDrag`, or read
`app.FileDrag()` in the view:

```go
giode.Config{
	OnFileDrag: func(d giode.FileDrag) {
		// d.Kind: FileDragEnter, FileDragOver, FileDragLeave, FileDragDrop
		// d.Position: window pixels, as layout uses them
	},
}

// In the view: the window redraws as the drag comes and goes.
if _, dragging := app.FileDrag(); dragging {
	zone = giode.Box(giode.Styles{BorderColor: "theme.primary", BorderWidth: 2}, zone)
}
```

| Platform | How |
| --- | --- |
| Windows | `WM_DROPFILES` for the drop, including when the app runs as administrator (giode lets the messages through Windows' privilege filter, which would otherwise silently block drops from a normal Explorer). With `OnFileDrag`, an OLE `IDropTarget` reports enter, over and leave too, except to an app running as administrator. |
| macOS | The view registers for file URLs (`registerForDraggedTypes`) and gets them from `performDragOperation:`. |
| Linux | XDND on X11 and `wl_data_device` on Wayland, through the giode CLI's Gio patches: Gio's own event loop reads those protocols' messages. A plain `go build` logs a warning once instead. |

Android, iOS and the web don't support it; a warning is logged once if it's set there.

## Files opened with the app

With `[[file_types]]` in `giode.toml`, the giode CLI puts the app in the desktop's "Open
with" for those files. A file opened that way arrives as a `file://` URL through
`Config.OnOpenURL` on macOS, and as the program's argument (`os.Args`) on Windows and
Linux; with a single-instance lock, a second launch's arguments reach the running instance
through `SingleInstanceLock.OnLaunchArgs`.

## Drawing video or 3D under the UI

`Config.Underlay` draws caller-rendered GPU content into each frame before the UI, without
copying it off the GPU: video frames from libmpv, a 3D or map engine. It runs on the
window's render thread with the window's GPU context current; the UI is then drawn over
its output without clearing, so only the UI's own backgrounds cover it (`Background`
isn't painted).

```go
app := giode.New(giode.Config{
	Renderer: giode.RendererOpenGL, // an OpenGL (ES) context on every platform
	Underlay: func(t giode.GPUTarget) {
		mpvRender(t.Framebuffer, t.Size.X, t.Size.Y) // flip Y: GL's origin is bottom left
	},
})
// From any goroutine, when the engine has a new frame:
app.Invalidate()
```

`GPUTarget` describes the frame's render target: `API` (`GPUOpenGL`, `GPUMetal`,
`GPUDirect3D11`, `GPUVulkan`), the OpenGL `Framebuffer` (0 is the window's), or the
`Texture` handle - the `MTLTexture` or `ID3D11RenderTargetView` - and the `Size` in
pixels. Engines with Metal or Direct3D renderers can keep Gio's defaults and draw into the
texture; commit their work before returning, since Gio draws with its own command queue.

`GetProcAddress` looks up the context's OpenGL functions by name, for engines that load GL
themselves - libmpv's `mpv_opengl_init_params.get_proc_address`, say - through ANGLE's
`libEGL.dll` on Windows, EGL on Linux and Android, and the OpenGL framework on macOS (it's
nil for other APIs). `Config.UnderlayRelease` runs on the render thread with the context
still current just before Gio releases its GPU resources: when the window closes, its view
goes away (an Android app leaving the screen), or the context is lost and made again, after
which `Underlay` runs in the new one. Free the engine's GPU objects there
(`mpv_render_context_free`).

```go
giode.Config{
	Renderer: giode.RendererOpenGL,
	Underlay: func(t giode.GPUTarget) {
		if mpv == nil {
			mpv = newMPVRenderContext(t.GetProcAddress)
		}
		mpv.render(t.Framebuffer, t.Size)
	},
	UnderlayRelease: func() { mpv.free(); mpv = nil },
}
```

With an OpenGL underlay, the MCP `screenshot` tool reads the finished frame back from the
GPU, so the video or 3D shows in it; with Metal or Direct3D it renders the UI alone.

`Renderer: giode.RendererOpenGL` asks for OpenGL everywhere:

| Platform | OpenGL |
| --- | --- |
| Linux, Android | OpenGL ES through EGL, Gio's default. |
| Windows | OpenGL ES through ANGLE: `libEGL.dll` and `libGLESv2.dll` must be next to the executable (see `[native]` in the giode CLI). |
| macOS, iOS | Gio's OpenGL renderer, built with the `nometal` tag: set `renderer = "opengl"` under `[app]` in `giode.toml`. |

Gio's OpenGL renderer saves and restores the GL state it changes around its frames, and
switches off scissor, depth and stencil tests and face culling an engine leaves on, so
engines don't need to clean up after themselves.

Both need giode's patched Gio, which `giode build` and `giode run` build against (adding
the `giode_gio` and `giode_gio2` tags); a plain `go build` gets an error from `Run` saying
so. `UnderlayRelease`, the screenshot read-back and `GetProcAddress`'s use with the release
need an up to date CLI (`giode upgrade`); with older ones `Run` returns an error naming the fix.

## Native window handle

`app.NativeHandle()` returns the native window handle: the `HWND` on Windows, the `NSView`
on macOS, the X11 window or `wl_surface` on Linux. `app.NativeView()` returns Gio's full
platform view event for other integrations. Both are zero until the window exists.
