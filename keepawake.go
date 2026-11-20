package giode

import (
	"context"

	"github.com/xmhrnx/giode/internal/platform/keepawake"
)

// KeepScreenOn keeps the screen from dimming and locking while on is
// set and the app is shown - for on-device generation the user is
// watching. On desktops it holds off the screen saver and display
// sleep.
func (a *App) KeepScreenOn(on bool) error {
	return keepawake.KeepScreenOn(context.Background(), a.host(), on)
}

// KeepAlive asks the system to keep the app running work it started -
// an agent run, a long download - until release is called:
//
//   - Desktops: the computer doesn't go to sleep while idle.
//   - iOS: the app gets the short grace period iOS allows after it
//     leaves the screen (about 30 seconds), via beginBackgroundTask.
//   - Android: a foreground service keeps the app running after it
//     leaves the screen, with an ongoing notification showing reason.
//     Call it while the app is shown. The app's manifest must declare
//     the service (see docs/platform.md), which gogio can't do yet;
//     without it KeepAlive returns an error wrapping ErrNotSupported,
//     so the app can tell the user to keep it open (with
//     KeepScreenOn) or save its progress in OnBackground.
//
// reason is shown where the platform shows one (Android's
// notification, Linux's inhibitor list, macOS's power assertions).
func (a *App) KeepAlive(reason string) (release func(), err error) {
	return keepawake.KeepAlive(a.host(), reason, func() {
		if askNotifications != nil {
			askNotifications(a)
		}
	})
}

// askNotifications, where set, asks for the notification permission
// once, for KeepAlive's notification (Android's foreground service).
var askNotifications func(*App)
