# FAQ

## What is Giode?

A UI library for Go built on [Gio](https://gioui.org). Gio draws everything itself on the
GPU and runs on Windows, macOS, Linux, Android, iOS and the browser; Giode adds what an app
needs on top: composable elements with CSS-like styles, a set of ready components, state,
routing, theming, platform services (notifications, files, tray, push, purchases) and a CLI
that packages it all.

## Is it ready for production?

Giode is in **alpha**. The apps on the [examples page](#/examples) ship on it, but the API
will still change between releases, sometimes in breaking ways. Pin a version in `go.mod`
and read the release notes when you upgrade.

## How is it different from Gio?

Gio is immediate mode with explicit layout: you write the drawing and the event handling
for every widget. Giode keeps Gio's model and speed but gives you elements styled like CSS
(`Padding`, `Gap`, `BorderRadius`, `BoxShadow`, flex layout), stateful components that
handle their own input, and app-level services. Raw Gio is always one `giode.Raw` away, and
every Giode element is a Gio widget.

## How does it compare to Fyne, Wails or a webview?

- **Wails, Tauri and Electron** render HTML in a webview. Giode draws natively on the GPU:
  no browser engine, a small binary, and the same rendering on every platform, phones
  included.
- **Fyne** also draws its own widgets, with its own fixed look. Giode is unopinionated
  about look and structure: every element takes styles, and components are plain Go.

## Which platforms does it support?

Windows, macOS, Linux (Wayland and X11), Android, iOS and the web (WebAssembly). The
[CLI](cli/index.md) builds packages for each: setup programs, MSIX, DMG, deb, rpm,
AppImage, Flatpak, APK, AAB, IPA and a PWA.

## Do I need the CLI?

Not on the desktop: a Giode app is a Go program that `go run` runs. The CLI is how you get
to phones, simulators, installers, signing and stores without a pile of tools and flags.

## Does it use cgo?

Gio uses cgo on Linux, macOS, iOS and Android (for the platform's windowing and GPU APIs),
not on Windows. Your own code only needs cgo if it links C libraries; the CLI bundles their
shared libraries into your packages.

## How fast is it?

Gio draws on the GPU and Giode builds only what the view returns each frame. `Memo`, `LazyList`
and `VirtualList` keep large or deep views fast; `Config.ShowFrameTimes` and the MCP
server's `frame_stats` show where the time goes. See
[Rendering performance](performance.md).

## Can an AI agent drive my app?

Yes. The built-in [MCP server](mcp.md) exposes the UI's semantic tree, input, screenshots
and named state, so an agent can test and operate the app.

## Can I draw video, 3D or a map under the UI?

Yes: `Config.Underlay` draws GPU content under the UI each frame without copying it off
the GPU. See [Window configuration](window.md#drawing-video-or-3d-under-the-ui).

## Can I host Giode in my own window or engine?

Yes. Import `giode/ui` (the same names, without the window layer) and call `ui.Frame` each
frame from your GLFW or engine loop. See [Low-level access](low-level.md).

## What's the license?

Apache 2.0, for the library and the CLI. Gio is MIT or Unlicense.

## Where do I report bugs or ask questions?

On GitHub: [issues](https://github.com/Mehran9675/giode/issues) for bugs and requests,
[giode-cli issues](https://github.com/Mehran9675/giode-cli/issues) for the CLI. Pull
requests are welcome; see [Contributing](contributing.md).
