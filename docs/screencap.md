# Screen capture

The `screencap` package captures a display as a stream of frames: for sharing the screen in
a call, recording it, or remote support.

```go
import "github.com/xmhrnx/giode/screencap"

srcs, err := screencap.Sources(ctx) // the displays
c, err := screencap.Start(ctx, app, screencap.Options{Source: srcs[0].ID, FPS: 15},
	func(f screencap.Frame) {
		encoder.Encode(f.Image) // *image.RGBA, the callback's to keep
	})
defer c.Stop()
```

Each `Source` has an `ID`, a `Name`, its `Bounds` on the desktop in pixels and whether it's
the `Primary` display. `onFrame` runs on its own goroutine; frames that arrive while it still
runs are dropped, so a slow encoder never builds a backlog. Call `app.Invalidate` from it to
show the frame.

| Option | |
|---|---|
| `Source` | a display's `ID`; the primary display when empty |
| `FPS` | frames a second, 10 by default |
| `MaxSize` | caps the frames' long side, scaling them down: 1920 by default, -1 for full size |

## Platforms

| | How | Permission |
|---|---|---|
| Windows | GDI, per monitor | none |
| macOS | ScreenCaptureKit (macOS 12.3 and later) | the user allows screen recording in System Settings, Privacy & Security; the first `Start` asks and returns `screencap.ErrPermissionDenied` until they do |
| Linux | X11, per monitor | none; on Wayland only windows on Xwayland show |
| Android | MediaProjection, held by a foreground service with an ongoing notification | Android asks the user on every `Start`; a refusal returns `ErrPermissionDenied` |
| iOS | ReplayKit: the app's own screen | iOS asks the user |

On Android the package declares its capture service and the `FOREGROUND_SERVICE_MEDIA_PROJECTION`
permission in its `android_manifest.xml`, which the Giode CLI merges into the app's manifest.
Only one capture runs at a time there. The web isn't supported; `Start` returns
`giode.ErrNotSupported`.

`examples/screenshare` picks a display and shows the capture live through a `video.Surface`
(see [Video](video.md)).
