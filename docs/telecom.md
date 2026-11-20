# System call screen

The `telecom` package puts an app's calls in the phone's own call system: the full-screen
incoming call screen, even over the lock screen; the call in the status bar and recents; and
answering, muting and hanging up from a headset, a watch or a car. iOS uses CallKit, Android a
self-managed `ConnectionService`. Computers have no system call screen: `Supported()` is
false there and the app shows its own (see `examples/call`).

The call's audio and video go through [rtc](rtc.md); `telecom` is only the ringing and the
controls.

```go
import "github.com/xmhrnx/giode/telecom"

telecom.Setup(app, telecom.Handler{
	OnAnswer: func(id string) { join(id) },          // the user answered
	OnEnd:    func(id string) { hangUp(id) },        // declined or hung up
	OnMute:   func(id string, m bool) { call.SetMuted(m) },
	OnAudio:  func(on bool) { /* iOS: start or stop the call's audio */ },
})

// Someone's calling (a message from your server):
telecom.ReportIncoming(ctx, telecom.Call{ID: id, Handle: "ada", Name: "Ada Lovelace"})

// The user calls someone:
id, _ := telecom.StartOutgoing(telecom.Call{Handle: "grace", Name: "Grace Hopper"})
// ... when they answer:
telecom.ReportConnected(id)

telecom.End(id)                          // hung up in the app
telecom.ReportEnded(id, telecom.RemoteEnded) // the other side hung up
```

Call IDs are UUIDs (`telecom.NewID()`); the same ID names the call to both the system and the
handler. Handler functions run on the app's goroutine.

## Ringing a sleeping app

An app that isn't running can't receive a WebSocket message. On iOS, the server sends a
**VoIP push** through APNs (the `voip` push type, to the token `OnVoIPToken` reports); iOS
wakes the app and giode reports the call to CallKit at once, as iOS requires. The push payload
carries the call: `id`, `handle`, `name` and `video` keys, or set `OnVoIPPush` to read your
own format.

On Android, send a high-priority FCM data message (see [push](platform.md)) and call
`ReportIncoming` from its handler.

## Setup

- iOS: set `app.calls = true` in `giode.toml` (the `voip` and `audio` background modes) and
  `ios.push` for VoIP pushes. Start the call's audio when `OnAudio(true)` arrives: CallKit
  activates the audio session itself.
- Android 8 and later: the package declares its connection service and the
  `MANAGE_OWN_CALLS`, `USE_FULL_SCREEN_INTENT` and `POST_NOTIFICATIONS` permissions in its
  `android_manifest.xml`, which the Giode CLI merges. Incoming calls show as a call
  notification (full screen over the lock screen) with Answer and Decline. Android 13 and later
  needs the notification permission granted; Android 14 and later asks the user to allow
  full-screen notifications in settings for apps that aren't the default dialer.
