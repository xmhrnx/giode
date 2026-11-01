# The giode CLI

The command-line tool for Giode apps: create a
project, run it on your computer, a phone, a simulator or the browser, and build signed
packages and installers for Android, iOS, macOS, Windows, Linux and the web.

```sh
giode new                  # asks for the platforms, name, ID and version, then creates the app
cd my-app                   # the folder you named
giode run                  # on this computer
giode run android          # on a connected phone: build, install, launch, follow the log
giode run windows          # build the Windows app and run it (with Wine off Windows)
giode build android windows-installer web
giode doctor               # what's installed, and what's missing for each target
```

It builds on [gogio](https://gioui.org/doc/install), Gio's packaging tool, whose code it
includes, and adds what gogio can't do: Android manifest entries (services, any
permission, raw XML), Info.plist keys (permission texts, background modes), Windows
installers, Authenticode signing, Linux desktop packaging, a project file instead of
long flag lists, and the run, doctor and device commands.

The CLI is optional - a Giode app is a plain Go program that `go run` runs on your
desktop - but it's how you get to phones, installers and stores without a pile of
platform tools and flags.

## Install

```sh tab="macOS / Linux"
curl -fsSL https://raw.githubusercontent.com/Mehran9675/giode-cli/main/install.sh | sh
```

```powershell tab="Windows"
irm https://raw.githubusercontent.com/Mehran9675/giode-cli/main/install.ps1 | iex
```

```sh tab="With Go"
go install github.com/mehran9675/giode-cli/cmd/giode@latest
```

The install scripts download the latest release for your system and put `giode` on your PATH
(`/usr/local/bin` or `~/.local/bin`; `%LOCALAPPDATA%\Programs\giode` on Windows). Set
`GIODE_VERSION=v0.2.0` for a specific release. `go install` puts `giode` in
`$(go env GOPATH)/bin`. `giode self-update` updates a release
install. Building apps needs Go 1.26 or later; `giode doctor` lists what else each
target needs.

Check it with:

```sh
giode version
giode doctor     # what this computer has, and what each platform still needs
```

## What builds where

| Target | Builds on | Needs |
| --- | --- | --- |
| `android`, `aab` | Linux, macOS, Windows | Android SDK with build tools, a platform and the NDK; a JDK |
| `ios`, `ios-sim`, `macos`, `dmg` | macOS | Xcode; for `ios`, a provisioning profile |
| `windows` | Linux, macOS, Windows | nothing extra for pure-Go apps (Gio doesn't need cgo on Windows); a cgo app needs a MinGW-w64 compiler and pkg-config for Windows: `windows.cc` |
| `msix`, `winget` | Linux, macOS, Windows | nothing extra for `msix` (giode writes it); `winget` builds the setup program, so NSIS |
| `windows-installer` | Linux, macOS, Windows | [NSIS](https://nsis.sourceforge.io) (`winget install NSIS.NSIS`, `brew install makensis`, `apt install nsis`) |
| `linux` | Linux | gcc and Gio's Wayland/X11/EGL development packages |
| `deb`, `rpm` | Linux | as `linux`; packages are written by giode itself |
| `appimage` | Linux | as `linux`, plus [appimagetool](https://github.com/AppImage/appimagetool/releases) on PATH |
| `flatpak` | Linux | as `linux`; with `flatpak-builder` a `.flatpak` bundle, without it the manifest |
| `web` | anywhere | nothing extra |

`giode doctor` checks all of these.

## Next

- [Commands](commands.md) - every command and its options.
- [giode.toml](config.md) - the project file.
- [Distribution](distribution.md) - signing, installers, native libraries, file
  associations, CI and Android libraries.
