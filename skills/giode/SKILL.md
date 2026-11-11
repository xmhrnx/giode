---
name: giode
description: Build, debug and ship cross-platform Go apps with the Giode UI library (github.com/xmhrnx/giode, on Gio) and the Giode CLI (`giode new/build/run/ci/release`, giode.toml). Use when code imports Giode or a project has a giode.toml, when the user asks for a Giode app, component, screen, layout, style, theme, platform feature (file pickers, notifications, tray, Android/iOS, Java bridge, elevation, i18n, RTL), MCP automation of a Giode app, or for building, signing and packaging one for Windows, macOS, Linux, Android, iOS or the web.
---

# Giode

Giode is a Go UI library on top of [Gio](https://gioui.org): no webview, GPU-rendered,
one codebase for Windows, macOS, Linux, Android, iOS and the web. UIs are composed from Go
functions returning `giode.Element`s, styled with CSS-like `giode.Styles`. The Giode CLI
(`github.com/xmhrnx/giode-cli/cmd/giode`) creates, builds, runs, signs and packages
apps from a `giode.toml`.

The full documentation is in `references/` next to this file (a copy of the repository's
`docs/`). Read the relevant page before writing code that uses a feature you haven't used
in this conversation; the API is alpha and changes. Map:

| Need | Read |
| --- | --- |
| First app, project layout | `references/getting-started.md`, `references/recipes.md` |
| Box/Row/Stack/Text/Table/Canvas/Responsive | `references/elements.md` |
| Every style property, colors, themes, `Mix` | `references/styles.md` |
| A component (Button, Dialog, Select, Menu, ...) | `references/components/<name>.md`; form controls in `input.md` and `controls.md` |
| Window options, title bar, tray, file dialogs | `references/window.md`, `references/components/titlebar.md` |
| Pickers, notifications, secure storage, elevation, i18n, RTL, push, purchases | `references/platform.md` |
| Android/iOS specifics, Java bridge | `references/mobile.md` |
| Goroutines, `Post`, `AfterLayout` | `references/threading.md` |
| Global state, routing | `references/state.md`, `references/routing.md` |
| Slow frames, `Memo`, long lists | `references/performance.md` |
| Custom components, portals, kit helpers | `references/extending.md` |
| Driving the running app from an agent | `references/mcp.md` |
| CLI commands, `giode.toml`, signing, CI, packaging | `references/cli/commands.md`, `references/cli/config.md`, `references/cli/distribution.md` |

## Mental model

- **The view function runs every frame.** `app.Run(func() giode.Element { ... })` rebuilds
  the element tree from plain Go state each frame. Plain variables are state; there's no
  virtual DOM or diffing.
- **Stateless elements are built every frame; stateful components are created once.**
  `giode.Box`, `Row`, `Stack`, `Text` are cheap values. Components that hold input or
  animation state (`giode.Button`, `giode.Input.Text`, `giode.Dialog`, `giode.Menu`,
  `giode.Scroll`, `giode.Tabs`, ...) are created once, outside the view (or per item with
  `giode.Keyed(scope, id, func() *T { ... })`), and laid out every frame. Recreating
  one each frame loses its state: clicks and typing stop working.
- **Everything UI runs on one goroutine.** The view, `Layout` methods and event handlers
  (`OnClick`, `OnChange`) run there. Background work hands results back with `app.Post(fn)`.
  Changing data the current frame is still reading (deleting the clicked row) goes in
  `giode.AfterLayout(fn)`. Blocking calls (`OpenFile`, `PickFiles`, `PickFolder`,
  `SaveFile`, `RestartElevated` prompts) must run in a goroutine, never in a click handler.
- **Lengths are dp.** `Width`, `Padding`, `FontSize`, `Gap`, `BorderRadius` are density
  independent; `Width: -1` / `Height: -1` fill the available space.
- **Raw Gio is always reachable**: `giode.Raw(func(gtx giode.Context) giode.Dimensions)` (`giode.Context`/`Dimensions` are Gio's layout types, so custom components need no Gio import),
  `app.RunRaw`, `app.Window()`.

## Minimal app

```go
package main

import (
	"fmt"
	"image"

	"github.com/xmhrnx/giode"
)

func main() {
	count := 0
	btn := giode.Button("Add", giode.Styles{Background: "theme.primary", Color: "theme.onprimary", BorderRadius: 8}).
		OnClick(func() { count++ }) // created once, outside the view

	app := giode.New(giode.Config{Title: "Counter", Size: image.Pt(480, 320), Resizable: true})
	if err := app.Run(func() giode.Element {
		return giode.Stack(giode.Styles{Width: -1, Height: -1, Align: giode.AlignCenter, Justify: giode.JustifyCenter, Gap: 12},
			giode.Text(fmt.Sprintf("Count: %d", count), giode.Styles{FontSize: 20}),
			btn,
		)
	}); err != nil {
		panic(err)
	}
}
```

## Layout and styling essentials

- `giode.Box(st, kids...)` is a flex container (`FlexDirection`, `Align` incl.
  `AlignStretch` for equal-height cards, `Justify`, `Gap`, `Wrap`, `FlexGrow`); `Row` and `Stack` are horizontal and vertical boxes.
- Absolute positioning: `Position: giode.PositionAbsolute` with `Top/Left/Right/Bottom`.
  An offset of 0 means unset; use `giode.Flush` for an explicit 0. `ZIndex` orders paint
  and input.
- **Zero means unset** throughout `Styles`. An all-zero `Padding` on a component falls back
  to its default and `Merge` keeps the base's; use `giode.NoInset` for an explicit zero.
- Insets: `giode.UniformInset(n)`, `giode.SymmetricInset(h, v)`, `giode.Inset{Top:, Right:,
  Bottom:, Left:}`; `Start`/`End` follow the reading direction (left/right, swapped right to left).
- Colors are CSS strings (`"#1e293b"`, `"rgb(...)"`, named) or theme tokens
  (`"theme.surface"`, `"theme.text"`, `"theme.primary/0.5"`) that follow `giode.SetMode`
  (light/dark) and `giode.SetPalette`. `""` / `"currentColor"` inherit the text color.
- Translucent colors blend in linear light and look brighter than in CSS. For tints use
  `giode.Mix(fg, bg, t)`: with a theme token it returns a `color-mix(...)` value that
  follows theme switches, so it can be computed once.
- Box shadows: `BoxShadow: giode.BoxShadow{Y: 8, Blur: 24, Color: "#00000066"}`.
- Custom drawing: `giode.Canvas(st, func(p *giode.Painter) { p.Arc(...); p.Sector(...);
  p.Path()... })` in dp, angles clockwise from 12 o'clock.
- Overlays: `Dialog`, `Sheet`, `Drawer` and `Menu` paint over the window and take no layout space;
  place them anywhere in the tree. Portals (`elements.RegisterPortal...`) render popups
  above everything.
- Responsive: `giode.Responsive(giode.Breakpoint{MaxWidth: 600, Name: "compact", View: ...}, ...)`
  and `giode.Compact()`.

## Components at a glance

`Button`, `IconButton`, `CopyButton`, `Input.Text/Checkbox/Slider/Select/Switch/Segmented/
AutoComplete`, `Chip`, `Badge`, `Avatar`, `Tabs`, `Accordion`, `Pagination`, `Progress`,
`Spinner`, `Tooltip`, `Menu` (context menus, menu bars, submenus), `Dialog`, `Sheet`,
`Drawer`, `Toaster`, `TitleBar` (frameless windows), `Scroll`, `VirtualList`, `LazyList`,
`TreeView`, `Swipeable`, `Transition`, `AppShell`, `Image`, `Icon` (Material, SVG,
IconVG), `Markdown`, `RichText`, `Table`, `Canvas`. Constructors and options are in
`references/components/`.

## Platform features (on `*giode.App` unless noted)

File dialogs `OpenFile`/`OpenFiles`/`SaveFile`/`PickFiles` (`PickedFile.Path` on desktops),
`PickFolder`, `PickPhoto`, `PickPhotos` (several, upright JPEGs), `PickMedia` (photos and videos as original files, the Photos picker on iOS), taking a photo (`takephoto.Take`) and saving to the gallery (`gallery.Save`) and dictation (`dictate.Start`) in opt-in packages so only apps using them carry the camera, photo library and microphone code,
`SaveToDownloads`, `OpenDownloads`, `OpenFolder`, `ReadClipboardImage`, `PlaySound` (with `EncodeWAV` for tones made in code); `Share` (`ShareFiles` for several);
receiving shares (`Config.OnShareReceived` with giode.toml's `app.share_target`); `Notify`; `OpenURL`; clipboard; `KeepAlive`,
`KeepScreenOn`; `Tray` (`SetCloseToTray`, `SetMinimizeToTray` at runtime); start at login
(`SetLaunchAtLogin`, `LaunchAtLogin`, `giode.LaunchedAtLogin()`, `Config.StartInTray`; MSIX
needs `[windows.msix] startup_task = true`); window control (`Minimize`, `Maximize`, `SetFullscreen`,
`SetAlwaysOnTop`, `MoveTo`, `Screen`, `RequestClose` - closes like the system button, honouring close-to-tray); `giode.IsElevated()` and `RestartElevated` (UAC,
pkexec, macOS admin prompt); single-instance lock (`Config.SingleInstanceLock`); context
actions in file managers; secure storage; push, in-app purchases and review prompts in
sub-packages; a live camera viewfinder with analysis frames, capture, torch and focus in
`camera`; on-device text recognition in `ocr`; fingerprint/Face ID in `biometric`; the print
dialog in `printing`; Android's Wi-Fi multicast lock in `multicast` (all in `references/platform.md`);
pinch-to-zoom and pan with `giode.Zoomable` (`references/components/zoom.md`); Android Java calls through `github.com/xmhrnx/giode/android`
(`android.CallBool(class, method, sig, args...)`).

## Translations and right-to-left

- Opt in with `Config.Intl: &giode.IntlOptions{Dir: "locales"}` (or `FS:` an embedded
  folder, required on Android, iOS and the web): `locales/en.json`, `fa.json`, ... mapping
  keys to text with `{placeholders}` or plural forms.
- `giode.Intl(key, data)` fills placeholders from a map, a struct or name/value pairs; a
  `count` picks the plural form. `giode.SetLocale("fa")` switches at runtime.
- Giode's own strings (window buttons, edit menu, ...) are translated for ~40 languages;
  override them with `giode.*` keys in your files.
- In a right-to-left locale the whole UI mirrors automatically. Use `Inset{Start:, End:}`,
  `giode.SideStart`/`SideEnd`, `giode.ToastBottomEnd` for direction-aware placement.
  Custom components: `kit.RTL(gtx)`, `kit.MirrorRTL`, `kit.GlyphChevronForward`.

## Code organization and clean code

Follow these rules for every Giode app and component you write or change. They apply on
top of the code's existing conventions; when a project already has a structure, keep it.

**One component per file.** Each component - a function returning an `Element`, a
stateful struct with a `Layout` method, or a fluent builder - lives in its own file named
after it in snake_case: `FileCard` in `file_card.go`, `SettingsDialog` in
`settings_dialog.go`. The component's private helpers, option types and constants stay in
its file; anything shared by several components moves to its own file (`theme.go`,
`format.go`). Don't add a second component to an existing file, even a small one; create
a new file.

**Project structure.** Separate the UI from the work it shows:

```
myapp/
  main.go            # giode.New, Config, app.Run: wiring only, no UI or logic
  giode.toml         # app metadata for the CLI
  assets/            # icons, images, fonts (embedded with //go:embed)
  locales/           # en.json, fa.json, ... (Config.Intl)
  internal/
    <domain>/        # the app's logic: no Giode imports, unit-tested on its own
  ui/
    app.go           # the root view and the top-level state it owns
    theme.go         # palette, shared colors, sizes and style helpers
    screens/         # one file per screen or page
    components/      # one file per reusable component
    platform/        # platform-specific code, split by build tag (_android.go, _windows.go, ...)
```

Smaller apps may flatten `ui/` into one package, but keep one component per file and
keep logic out of the UI package.

**Clean code.**
- Views are pure: a view function reads state and returns elements. No file or network
  I/O, no blocking calls and no state changes while building the tree; do the work in event
  handlers or goroutines and hand results back with `app.Post`.
- Create stateful components once (in a constructor such as `newSettingsDialog(app)`) and
  keep them in the struct that owns them; never in the view.
- Keep functions short and single-purpose: split a view longer than about 40-60 lines into
  named components.
- Name things for what they are and do (`saveRecoveredFiles`, `fileRow`), not how they're
  built; exported names get doc comments that say what they do, and comments explain why,
  not what.
- Put styles that repeat in `theme.go` (theme tokens, `giode.Mix` tints, shared `Styles`
  values) instead of copying literals; use theme tokens rather than hard-coded colors so
  light and dark modes work.
- Handle every error: show it to the user (toast, dialog, inline message) or return it;
  never drop it silently.
- No dead code, commented-out code or unused parameters; no global mutable state except
  `giode.State` values meant to be shared.
- Platform-specific code goes behind build tags in separate files, with a stub for the
  other platforms, never `runtime.GOOS` switches scattered through views.
- Run `gofmt`, `go vet` and `go test ./...` before finishing; test logic in `internal/`
  packages, and UI behavior with laid-out elements and semantics (see "Inspecting a running
  app").

## Inspecting a running app

Start any Giode app with `GIODE_MCP=1` (or `Config.MCP: true`): an MCP server on
`http://127.0.0.1:7600/mcp` (`MCPOptions.Host`/`Port`, `GIODE_MCP_PORT`) exposes the UI
tree, screenshots, clicks, typing and scrolling. Use it to check a change visually instead
of guessing. For headless tests, lay elements out with a `layout.Context` and read
`elements.BeginSemantics()`/`EndSemantics()` node bounds; `gioui.org/gpu/headless` renders
pixels where a GPU (or Mesa with `EGL_PLATFORM=surfaceless`) is available.

## The CLI

Install: `go install github.com/xmhrnx/giode-cli/cmd/giode@latest` (or the install
scripts in the Giode CLI README).

| Command | What it does |
| --- | --- |
| `giode new [dir]` | Scaffold a project (asks for platforms, id, name; `-y` skips) |
| `giode run [target]` | Build and run: desktop by default, `android` (device), `ios-sim`, `web` |
| `giode build [target...]` | Build into `dist/<platform>`; no target builds this computer's platform. Targets: `android`, `aab`, `ios`, `ios-sim`, `macos`, `dmg`, `windows`, `windows-installer`, `msix`, `winget`, `linux`, `deb`, `rpm`, `appimage`, `flatpak`, `web`, `desktop`, `all`. `-release` signs |
| `giode doctor` | What this computer is missing per platform, with install commands |
| `giode ci` | Write `.github/workflows/release.yml` from `[ci]` in giode.toml; rerun with `-f` after changing `[ci]` instead of editing the workflow |
| `giode release` | Update feed for desktop auto-update |
| `giode init` | Add a giode.toml to an existing Go module |
| `giode cert`, `giode keystore`, `giode sign` | Windows/macOS certificates, the Android keystore, signing a file |
| `giode bump`, `giode manifest`, `giode devices`, `giode licenses`, `giode links` | Version bumps, manifest and Info.plist preview, connected devices, open-source notices, app-link files |
| `giode upgrade`, `giode clean`, `giode version` | Update the CLI, remove build caches, print the version |

`giode.toml` holds `[app]` (name, id, version, build, icon, platforms, tags, ldflags),
`[android]` (permissions, keystore, `icon_foreground`/`icon_background`/`icon_monochrome`,
`java` sources, `application_attributes`, dependencies), `[ios]`, `[macos]`, `[windows]`,
`[linux]`, `[web]`, `[native]` (shared libraries bundled per OS), `[update]` (desktop
auto-update), `[privacy]`, `[[context_actions]]`, `[[file_types]]`, `[ci]` (`trigger`,
`test`, `setup`, `arch`, `windows_msys2`). The CLI builds against a patched copy of Gio (window fixes, Android frame
pacing); `references/cli/distribution.md` lists the patches. Read `references/cli/config.md`
for every key before editing giode.toml.

## Gotchas (each cost real debugging time)

- A stateful component recreated inside the view loses its state; create it once or use `Keyed`.
- App-wide shortcuts belong in `giode.Shortcuts(giode.Shortcut("Mod+T", fn), ...)` (or
  `Config.OnKey` / `giode.OnKey` for raw keys), which only see keys nothing else took; a catch-all `key.Filter` in the view steals keys from fields and dialogs.
- Blocking dialogs called from a click handler deadlock; call them from a goroutine and `Post` the result.
- `Padding: UniformInset(0)` is "unset"; use `giode.NoInset`.
- Offsets of 0 are unset; use `giode.Flush`.
- Mixed colors computed once from fixed hex values don't follow theme switches; mix theme tokens.
- Translucent overlays look too bright; use `Mix` for opaque tints.
- Text uses the embedded Go fonts by default (Latin, Greek and Cyrillic), falling back to
  system fonts; register a font with `giode.RegisterFont` for Arabic, Hebrew, CJK, ...
- Wrapping a component in a `widget.Clickable` clips what it paints, shadows included; use
  `kit.LayoutClickable` or `giode.Pressable`.
- On Android and iOS there are no folders beside the app: embed translations and assets
  with `//go:embed`.
- The CLI's Android icon: without `icon_foreground`, `app.icon` is shrunk into the 66dp
  safe zone; draw your own layers for full control.
