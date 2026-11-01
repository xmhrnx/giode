# Getting started

Giode apps are plain Go programs. This page takes you from nothing to a running app on
your computer, then on a phone, in about five minutes.

> Giode is in **alpha**. The API is still taking shape and will change between releases; pin a
> version in `go.mod` and read the release notes when you upgrade.

## 1. Install Go

Giode needs **Go 1.26 or later**. Get it from [go.dev/dl](https://go.dev/dl), then check:

```sh
go version
```

On Linux, Gio (which Giode draws with) also needs a C compiler and the Wayland/X11
development headers:

```sh tab="Debian / Ubuntu"
sudo apt install gcc pkg-config libwayland-dev libx11-dev libx11-xcb-dev libxkbcommon-x11-dev libgles2-mesa-dev libegl1-mesa-dev libffi-dev libxcursor-dev libvulkan-dev
```

```sh tab="Fedora"
sudo dnf install gcc pkg-config wayland-devel libX11-devel libxkbcommon-x11-devel mesa-libGLES-devel mesa-libEGL-devel libXcursor-devel vulkan-headers
```

```sh tab="Arch"
sudo pacman -S gcc pkgconf wayland libx11 libxkbcommon-x11 mesa libxcursor vulkan-headers
```

## 2. Install the giode CLI

The [giode CLI](cli/index.md) creates projects, runs them on your computer, a phone, a
simulator or the browser, and builds signed packages and installers.

```sh tab="macOS / Linux"
curl -fsSL https://raw.githubusercontent.com/Mehran9675/giode-cli/main/install.sh | sh
```

```powershell tab="Windows"
irm https://raw.githubusercontent.com/Mehran9675/giode-cli/main/install.ps1 | iex
```

```sh tab="With Go"
go install github.com/mehran9675/giode-cli/cmd/giode@latest
```

Then let it check your computer:

```sh
giode doctor
```

It lists what's installed and, for each platform, what's missing with the command to
install it. Android needs the Android SDK, NDK and a JDK; iOS and macOS builds need a Mac
with Xcode.

## 3. Create an app

```sh
giode new
```

`giode new` asks a few questions. First, which platforms the app is for: every platform
is listed, whether or not this computer can build it. Move with the arrow keys, check
with **Space**, continue with **Enter**:

```text
? Which platforms is the app for?
  ↑/↓ move · space check · a all · enter continue
  [ ] Windows  builds anywhere; NSIS for the setup program
  [x] macOS  builds on a Mac with Xcode
  [ ] Linux  builds on Linux
❯ [x] Android  needs the Android SDK and NDK
  [ ] iOS  builds on a Mac with Xcode
  [ ] Web  builds anywhere (WebAssembly)
```

Then the folder, app name, app ID, version, Go module path, publisher, description and
website, each with a suggestion in parentheses that **Enter** accepts:

```text
? App name (Notes)
? App ID (reverse-DNS, can't change after publishing) (com.example.notes) com.acme.notes
? Version (0.1.0)
```

You get a counter app in `main.go`, `go.mod` with the latest Giode, `giode.toml` set up
for the platforms you picked, a placeholder icon and a README. Skip the questions with
`giode new -y -platforms android,ios notes`.

## 4. Run it

```sh
cd notes
giode run              # on this computer
giode run android      # on a connected phone or emulator: builds, installs, launches, shows the log
giode run ios-sim      # in the iOS simulator (macOS)
giode run web          # in the browser
```

## 5. Read the code

`main.go` is a complete app. Here's the same idea in its smallest form:

```go
package main

import (
	"fmt"
	"image"

	"github.com/mehran9675/giode"
)

func main() {
	giode.UseGofont() // embed a font for text

	count := 0
	btn := giode.Button("Click me",
		giode.Styles{
			Background:   "#3b82f6",
			Color:        "#ffffff",
			BorderRadius: 8,
		},
	).OnClick(func() { count++ })

	app := giode.New(giode.Config{
		Title:      "Hello",
		Size:       image.Pt(800, 600),
		Resizable:  true,
		Background: "#0f172a",
	})

	if err := app.Run(func() giode.Element {
		return giode.Stack(giode.Styles{
			Align:   giode.AlignCenter,
			Justify: giode.JustifyCenter,
			Width:   -1,
			Height:  -1,
		},
			giode.Text(fmt.Sprintf("Clicks: %d", count), giode.Styles{Color: "#f8fafc"}),
			btn,
		)
	}); err != nil {
		panic(err)
	}
}
```

Everything on screen is an `Element`, built from Go functions and styled with CSS-like
`Styles`. The view function runs every frame; plain Go variables are your state.

## 6. Build for release

```sh
giode build                       # this computer's platform
giode build -release android aab  # signed APK and Play bundle
giode build windows-installer dmg deb
```

Packages land in `dist/<platform>`. [Distribution](cli/distribution.md) covers signing,
installers and store uploads; `giode ci` writes a GitHub Actions workflow that builds and
publishes them for every tag.

## Without the CLI

The CLI is optional on the desktop. In any Go module:

```sh
go get github.com/mehran9675/giode@latest
go run .
```

## Where next

- [Recipes](recipes.md): the different ways to write components with Giode.
- [Elements](elements.md) and [Style properties](styles.md): the building blocks.
- [Components](components/README.md): buttons, inputs, lists, dialogs, menus and more.
- [Examples](#/examples): apps built with Giode.

## The two kinds of elements

Immediate mode rebuilds the UI every frame. Giode handles the two consequences explicitly:

- **Stateless elements** (`Box`, `Row`, `Stack`, `Text`, `Divider`, `Responsive`, `Image`,
  `Progress`, `Spinner`, icons) are cheap to build inside the view function every frame.
- **Stateful components** (`Button`, `Input.Text`, `Input.Checkbox`, `Input.Slider`,
  `Input.Select`, `Tabs`, `Scroll`, `Drawer`, `Dialog`, `Menu`, `Router`) hold their
  persistent state. Create them **once, before `Run`**, and lay them out every frame:

```go
name := giode.Input.Text("Name")  // once
...
app.Run(func() giode.Element {    // every frame
	return giode.Stack(giode.Styles{}, name)
})
```

Creating a stateful component inside the view function resets it every frame. The same
applies to elements with hover styles: `Box(...).OnHover(...)` stores hover state in the
element, so create it once and reuse it.

## Lifecycle

`Run` blocks for the lifetime of the app and terminates the process when the window
closes. Hook into the lifecycle with `Config.OnStartup`, `OnReady`, `OnShutdown` and
`OnBeforeClose`; see [Window configuration](window.md).
