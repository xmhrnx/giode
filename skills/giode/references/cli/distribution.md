# Distribution

Signing, installers, bundled libraries, file associations, CI and the patched Gio the CLI builds with.

## Signing

`giode build -release` builds for distribution and signs everything it can with the
certificates in `giode.toml`; without `-release`, Android uses the debug key and the
rest are unsigned.

| Platform | What signs it | Set up |
| --- | --- | --- |
| Android | `android.keystore` | `GIODE_KEYSTORE_PASS=... giode keystore` creates one. Keep it and its password safe: Google Play only accepts updates signed with it (with Play App Signing, it's your upload key). |
| Windows | `windows.signing`: an Authenticode `.pfx`, or a `command` | Signs the executable and the setup program, with a timestamp so signatures outlive the certificate. Uses `signtool` on Windows and `osslsigncode` elsewhere. For keys in a hardware token or a cloud service (Azure Trusted Signing, a KMS), give `command` with `{file}` where the file goes. `giode cert` makes a self-signed certificate to test with; for public releases, use one from a certificate authority so SmartScreen doesn't warn. |
| macOS | `macos.identity` (or `macos.profile`) | A "Developer ID Application" identity in your keychain. With `macos.notary_id` and the app-specific password in `$GIODE_NOTARY_PASS`, the app is also notarized, so Gatekeeper opens it without warnings. |
| iOS | `ios.profile` | Device builds always need a provisioning profile; the matching identity must be in your keychain. |

`giode sign <file>...` signs other Windows files the same way.

## Windows installer

`giode build windows-installer` makes `dist/windows/<App>-<version>-<arch>-setup.exe` with NSIS:
a wizard (welcome, optional license, folder, progress, finish with a "Run" box), a Start
menu shortcut (and a desktop one with `desktop_shortcut`), an entry in Settings > Apps
with publisher, version and size, and an uninstaller. By default it installs for the
current user into `%LOCALAPPDATA%\Programs`, without an administrator prompt;
`per_machine = true` installs for everyone into Program Files. It runs silently with
`/S`. The uninstaller isn't signed yet.

## Native libraries

A cgo app linking shared libraries (libmpv and the dozens it brings) needs them in its
packages. With `[native] bundle = true`, giode reads the built binary's imports
(PE, ELF or Mach-O, recursively, so it works when cross-building), leaves out the
system's (Windows' DLLs, macOS' `/usr/lib` and `/System`, a Linux desktop's C library,
graphics, windowing and sound stacks) and bundles the rest:

| Target | Where they go |
| --- | --- |
| `windows`, `windows-installer`, `msix`, `winget` | next to the `.exe`, installed (and uninstalled) with it |
| `macos`, `dmg` | `Contents/Frameworks`, renamed `@rpath/<name>` with every reference rewritten and the executable given the run path (as `dylibbundler` does), signed with the app |
| `linux`, `appimage`, `flatpak`, `deb`, `rpm` | `lib/` next to the binary, or the package's `lib/<command>`; the binary's run path finds them |

```toml
[native]
bundle = true
exclude = ["libvulkan*"]              # never bundled

[native.windows]
search = ["$MSYSTEM_PREFIX/bin"]      # where the DLLs are (MSYS2, a MinGW sysroot)
libraries = ["$MSYSTEM_PREFIX/bin/libEGL*.dll"]  # also bundled

[native.macos]
search = ["/opt/homebrew/lib"]

[native.linux]
system = true                         # .deb and .rpm depend on the system's packages instead
deb_depends = ["libmpv2"]             # the .rpm requires the libraries' sonames
libraries = ["/usr/lib/x86_64-linux-gnu/libva-drm.so.2"]  # also bundled: plugins loaded at run time
```

Paths in `search` and `libraries` may use environment variables (`$VAR`, `${VAR}`,
`%VAR%`), `~/` and globs; relative paths are from giode.toml's folder. A `libraries`
pattern that matches nothing is an error; a `search` folder that doesn't exist is skipped.
Windows' imports include delay-loaded DLLs, and the Visual C++ runtime
(`vcruntime140.dll`, `msvcp140.dll` and the rest) is bundled when found but not warned
about when it isn't.

With `renderer = "opengl"`, Windows builds need ANGLE's `libEGL.dll` and
`libGLESv2.dll`, which the binary loads at run time rather than imports. giode bundles
them without `[native]`: it looks in `[native.windows] search`, `$MSYSTEM_PREFIX/bin`,
MSYS2's `clangarm64`, `clang64`, `ucrt64` and `mingw64` (under `$MSYS2_ROOT` or
`C:\msys64`) and Visual Studio Code's install, taking the first pair built for the target
CPU, along with the DLLs they import. `pacman -S mingw-w64-x86_64-angleproject` (or
`mingw-w64-clang-aarch64-angleproject`) installs them.

Libraries a binary imports that aren't the system's and aren't found are listed in a
warning; on Windows that leaves out DLLs in `System32` (when building on Windows) and the
graphics driver's (`vulkan-1.dll`, `OpenCL.dll`). A program's own folder - the output
folder, with the last build's copies - isn't searched for its DLLs, so a rebuild takes
them from the search folders again. `giode doctor` checks the tools: `install_name_tool` and `codesign` on macOS,
`windows.cc` and pkg-config for Windows cgo builds.

## File associations

`[[file_types]]` puts the app in "Open with" for those files on every desktop:

```toml
[[file_types]]
name = "MKV video"
extensions = ["mkv", "mk3d"]
mime_types = ["video/x-matroska"]

## CI for cgo apps

`giode ci`'s workflow installs what Gio needs; a cgo app's C libraries come from
`[ci] setup`, commands run on each runner before building. `windows_msys2` builds the
Windows packages on a Windows runner in MSYS2's MINGW64 environment, where libraries like
libmpv are a `pacman` away, instead of cross-building them on Linux:

```toml
[ci]
windows_msys2 = true

[ci.setup]
macos = "brew install mpv"
linux = "sudo apt-get install -y libmpv-dev"
windows = "pacman -S --noconfirm ${MINGW_PACKAGE_PREFIX}-mpv"
```

`arch` builds each desktop's packages for more CPUs. With `arm64`, Linux packages are
also built on `ubuntu-24.04-arm`, and Windows ones cross-built for arm64 (or, with
`windows_msys2`, built on `windows-11-arm` in MSYS2's CLANGARM64 environment, where
`$MINGW_PACKAGE_PREFIX` is `mingw-w64-clang-aarch64`). macOS' dmg is universal already.

```toml
[ci]
arch = ["amd64", "arm64"]   # default ["amd64"]
```

The workflow builds the platforms in `app.platforms` (without it, all but iOS). An iOS job
builds an IPA when the repository has the `IOS_CERT_BASE64`, `IOS_CERT_PASS` and
`IOS_PROFILE_BASE64` secrets (`IOS_TEAM_ID` sets `ios.team_id`), and checks the simulator
build otherwise; the profile reaches the build through `$GIODE_IOS_PROFILE`, so giode.toml
needn't name it. A test job runs first and stops the release when it fails. Change what it
runs, or turn it off, with `test`. `trigger = "release"` builds when a GitHub release is
published, adding the files to it, instead of on a pushed tag:

```toml
[ci]
trigger = "release"                    # default "tag"
test = "go vet ./... && go test ./..." # default "go test ./..."; "" runs none
```

Rerun `giode ci -f` after changing `[ci]` rather than editing the workflow: it's written
from giode.toml and overwritten.

Every package's name carries its CPU, so builds for two CPUs can share a release:
`<App>-<version>-<arch>-setup.exe`, `<App>-<version>-<arch>.msix` and
`<App>-<version>-universal.dmg` (or `-arm64`/`-amd64` with a single `-arch`).

## Android icon

Android 8 and later draw the launcher icon from two layers, each a 108dp square of which
the launcher shows only the middle, cut to its own shape (a circle, a squircle, ...). Keep
the artwork inside the middle 66dp. Without settings, giode builds them from `app.icon`: the
icon shrunk into that safe zone on a transparent foreground, over the color around its edge
(white for a transparent edge), so a square icon isn't clipped. To draw the layers
yourself:

```toml
[android]
icon_foreground = "assets/android-foreground.png" # the whole 108dp canvas, transparent around the artwork
icon_background = "assets/android-background.png" # or a color: "#0f172a"
icon_monochrome = "assets/android-monochrome.png" # optional: its shape is tinted for themed icons (Android 13+)
```

Without `icon_monochrome`, themed icons show the full-color icon. Android 7 and earlier
use `app.icon` as is.

## Android libraries

Android builds take libraries from Maven repositories, as Gradle would:
`android.dependencies` lists `group:artifact:version` coordinates, which
giode resolves with their dependencies from Google's Maven repository and
Maven Central (cached in the user cache directory), and
`android.libraries` adds local `.aar` and `.jar` files. Their classes are
dexed with the app's (multidex when needed), their resources linked with
R classes for their packages, their manifests merged into the app's, and
their native libraries (for the app's ABIs) and assets packaged.

A Go package can ask for libraries itself: an `android_dependencies.txt`
next to its Go files lists coordinates, and an `android_manifest.xml`
adds permissions and `<application>` components, for every app importing
it. That's how giode's `push` (Firebase Cloud Messaging), `store` (Play
Billing) and `review` (Play In-App Review) packages bring what they need;
`push` also needs `android.google_services`, the `google-services.json`
file from the Firebase console, which giode turns into the resources
Firebase reads, as Google's Gradle plugin does.

## Patched Gio

`giode build` and `giode run` build against a copy of your Gio version with giode's
patches, adding the `giode_gio` and `giode_gio2` build tags:

- **Underlay hooks**, on every platform: what giode's `Config.Underlay`,
  `Config.UnderlayRelease` and `Config.Renderer` use to draw video or 3D under the UI. The
  window calls the hook with the frame's render target before drawing the UI over it
  without clearing, calls another once the UI is drawn (the MCP screenshot reads the frame
  back there) and a third before releasing its GPU resources; Gio's OpenGL renderer saves
  and restores the GL state around its frames, and on Windows it can prefer OpenGL ES
  (ANGLE) over Direct3D. `renderer = "opengl"` under `[app]` adds the `nometal` tag to
  macOS and iOS builds, for Gio's OpenGL renderer there.
- **File drags on Linux**: XDND on X11 and `wl_data_device`'s drag and drop on Wayland,
  for `Config.OnFileDrop` and `OnFileDrag`. Gio's event loop reads those protocols'
  messages, so they can't be handled outside it.
- **Minimized windows on Windows**: Gio took a minimized window for a restored one, so
  `WindowMinimized`, desktop `OnBackground` and the way back from them never came.
- **Fullscreen on Windows**: Gio showed a fullscreen window maximized without its frame,
  and Windows draws the taskbar over maximized windows. The patched copy makes it a popup
  covering the monitor, and puts it back where and as it was when leaving fullscreen.
- **Fullscreen on macOS**: Gio wrote the window's style mask back after starting AppKit's
  fullscreen transition, setting the fullscreen bit again in the middle of leaving
  fullscreen, and resized the window during the transition; the window's mode was read
  again only when it drew, so a window that wasn't drawing kept its old mode. The patched
  copy leaves the fullscreen bit to AppKit, doesn't resize during transitions, reports the
  mode when a transition ends, and puts a window back to the normal level when `TopMost`
  is turned off.
- **Input from automation**: the MCP server's `ui_pointer` and `ui_key` deliver pointer
  and key events to the window's input as the platform would.

And two Android fixes Gio hasn't released:

- **Full refresh rate.** Gio asks Android's Choreographer for the next frame only after
  presenting the current one. Presenting waits for a vsync, so the request misses the next
  one, and a 120Hz display gets 60 frames a second while anything animates. The patched
  copy asks before drawing, and drops a frame callback that arrives after the animation
  stopped instead of drawing a frame for nothing.
- **Held keys repeat.** After every edit Gio restarts the soft keyboard's input connection
  if the editor's snippet of text differs from the keyboard's - which it does after any
  edit, since the editor's snippet window moves with the text. The restart cancels the
  keyboard's key repeat, so holding backspace deleted one character. The patched copy
  restarts only when the text changed under the keyboard (an app clearing or replacing the
  field), not when the editor just moved its window over text the keyboard already has.

Your `go.mod` isn't touched. The copy is made once per Gio version in the user cache
directory (`giode/gio/<version>-giode<n>`, made again when the patches change), and the build gets a temporary `go.mod` with
`replace gioui.org => <copy>`, passed with `-modfile` to the `go build` and `go list`
commands run in your module (`GOFLAGS` is left alone). The patch is skipped, with a note,
when go isn't in module mode, or your module already replaces `gioui.org`, vendors its
dependencies or is in a Go workspace, or when your Gio version doesn't have the code the patch expects. Set
`gio_patches = false` under `[app]` to build against plain Gio (or under `[android]` for
Android builds only).

