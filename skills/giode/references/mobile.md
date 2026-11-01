# Mobile (Android)

A giode app runs on Android unchanged: the same `giode.New(cfg).Run(view)` builds into an
APK with Gio's `gogio` tool. This page covers the build, what behaves differently on a
phone, and the APIs made for it. iOS builds work the same way through `gogio -target ios`,
but fewer of the mobile APIs below are implemented there yet.

## Building

The [giode CLI](https://github.com/mehran9675/giode-cli) wraps everything below:
`giode build android` reads the app's name, ID, version, icon and permissions from
`giode.toml`, `giode run android` builds, installs and launches on a connected device,
and `giode doctor` checks the tools listed here. It also adds what gogio can't: the
foreground service `KeepAlive` needs, any manifest permission, and Info.plist keys on
iOS. The rest of this section uses gogio directly.

Install the tools once:

- **gogio**: `go install gioui.org/cmd/gogio@latest`
- **Android SDK** with the build-tools and a platform (Android Studio's SDK manager, or
  `sdkmanager "platforms;android-35" "build-tools;35.0.0"`). Point `ANDROID_HOME` at it.
- **Android NDK** (`sdkmanager "ndk;27.2.12479018"`). gogio finds it under
  `$ANDROID_HOME/ndk`; set `ANDROID_NDK_ROOT` if it lives elsewhere.
- **A JDK** (17 or later), for `keytool` and the dexer. gogio uses `JAVA_HOME` or the
  `keytool` on your `PATH`.

Then build from your module:

```sh
gogio -target android -appid com.example.sonora -name Sonora -version 1.2.0.12 \
      -icon appicon.png -o sonora.apk .
adb install -r sonora.apk
```

- `-appid` is the Android package name; keep it stable, it identifies the app.
- `-version` is `major.minor.patch.versioncode`; increase the version code for every
  release you upload.
- `-minsdk` / `-targetsdk` set the API levels (gogio's defaults are fine for testing; the
  Play Store requires a recent `-targetsdk`).
- `-arch arm64` builds a single architecture, which is faster while developing.

**Signing.** Without `-signkey`, gogio signs with `~/.android/debug.keystore`, creating it
if needed - fine for testing, not for the Play Store. For a release, create a keystore once
and pass it:

```sh
keytool -genkey -keystore release.jks -alias sonora -keyalg RSA -validity 10000
gogio -target android -appid com.example.sonora -signkey release.jks -signpass "$PASS" -o sonora.apk .
```

Keep the keystore safe: updates to an installed app must be signed with the same key.

## What the platform ignores

Android gives the app the whole screen, so these `Config` fields have no effect there:

| Field | Why |
| --- | --- |
| `Size`, `MinSize`/`MaxSize`, `MinWidth`/`MinHeight`/`MaxWidth`/`MaxHeight`, `Resizable` | The window is the screen. Use `Responsive` breakpoints to adapt the layout. |
| `Frameless`, `Draggable`, `Center`, `AlwaysOnTop`, `WindowStartState` and the start-state flags | There's no desktop window to decorate or place. |
| `Icon` | The launcher icon comes from gogio's `-icon` at build time. |
| `Title` | The label under the icon comes from gogio's `-name`. |
| `SingleInstanceLock` | Android already runs one instance per app. |
| `OnFileDrop` | Windows only. Use `OpenFile` instead. |

Fields that matter more on a phone: `StatusBar`/`NavigationBar` (system bar colors, by
default `Background`, following theme changes), `Orientation`, `OnBackground` and
`OnForeground` - see [window.md](window.md).

## Permissions

Permissions are declared by importing Gio's permission packages; gogio adds the matching
entries to the manifest:

```go
import (
	_ "gioui.org/app/permission/camera"
	_ "gioui.org/app/permission/microphone"
)
```

Available: `bluetooth`, `camera`, `microphone`, `networkstate`, `storage`, `wakelock`. The
`INTERNET` permission is added whenever the program imports `net`, which every giode app
does (for the MCP server). Permissions Android rates "dangerous" (camera, microphone,
location) also need a runtime prompt, which Gio doesn't show for you: request it from the
activity, reachable through `App.NativeView()` (an `app.AndroidViewEvent` whose view's
`getContext()` is the activity).

## Screen insets

The status bar, navigation bar and display cutouts are handled by Gio: the view is laid out
inside the safe area, and `Config.Background` fills behind the bars. There's nothing to add.

## Storage

Apps can't read or write the user's files by path. Use:

- `app.DataDir()` (from `gioui.org/app`) for the app's own files - settings, drafts,
  caches. It's private to the app and removed with it. Call it from `main` or later.
- `App.OpenFile`, `OpenFiles` and `SaveFile` to let the user pick a file through the system
  picker - see [window.md](window.md#opening-and-saving-files).
- `App.Share` to hand a file or text to another app through the share sheet - see
  [window.md](window.md#sharing).

Save anything the user would hate to lose in `Config.OnBackground`: Android may end a
backgrounded app without warning, and `OnShutdown` doesn't run then.

## Calling Java

The `github.com/mehran9675/giode/android` package calls static Java methods through the
app's class loader, so the app's own classes resolve as well as the SDK's. Put the compiled
classes in a `.jar` in one of the app's packages (giode build adds it to the APK; with
`android.java` in giode.toml it compiles the sources for you):

```go
import "github.com/mehran9675/giode/android"

ok, err := android.CallBool("com/example/usb/Usb", "hasPermission",
	"(Landroid/content/Context;Ljava/lang/String;)Z", android.Context(), deviceName)
```

`CallVoid`, `CallBool`, `CallInt`, `CallString` and `CallBytes` take Go strings, byte
slices, numbers, bools and object references as arguments; `Java(class, fn)` runs `fn`
with the JNI environment (from `git.wow.st/gmp/jni`) for anything else. Java calls back
through `//export Java_<class>_<method>` functions, with `android.GoString` and
`android.GoBytes` for their arguments. Elsewhere the calls return `errors.ErrUnsupported`.

## Touch and the Back button

giode adapts to touch on its own: on a touch screen `Button`, `IconButton` and `Checkbox`
grow their tap area to 48dp, tooltips appear on a long-press, and a press slid off a
control isn't a click (see [extending.md](extending.md#touch-input)). The Back button
closes the topmost `Dialog`, `Drawer`, `Menu` or `Select` sheet, then goes back through a
`Router`'s history, then leaves the app; register your own handler with
`giode.HandleBack` (see [window.md](window.md#back-button)).

Phone-friendly layout helpers:

- `Responsive` with `MaxWidth` in dp, and `giode.Compact()` inside the chosen view - see
  [elements.md](elements.md#responsive).
- `Input.Text(...).Keyboard(giode.KeyboardEmail)` picks the on-screen keyboard.
- `Input.Select(...).Sheet(giode.SheetCompact)` opens its options as a bottom sheet on
  narrow screens.

## Testing phone layouts on a desktop

Run the app with the MCP server (`GIODE_MCP=1`) and resize the window to a phone's size:

```json
{"name": "window", "arguments": {"action": "resize", "width": 400, "height": 840}}
```

or call `app.SetSize(400, 840)` yourself. Desktop and phone share the dp units, so the
layout, breakpoints included, matches what the phone shows; touch behavior can be checked
on a touch screen or a device.
