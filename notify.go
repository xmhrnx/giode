package giode

import (
	"context"
	"errors"
)

// ErrNotificationsOff is returned by Notify when the user turned the
// app's notifications off (or didn't allow them).
var ErrNotificationsOff = errors.New("giode: notifications are turned off for this app")

// Notify shows a system notification with title and body - to tell
// the user a transfer arrived or a download finished. It works on
// Android, iOS, macOS, Windows and Linux.
//
// On Android 13 and later, iOS and macOS, the app needs the user's
// permission. The first Notify of a run asks for it when it's missing,
// and shows its notification if the user allows; after that Notify
// returns ErrNotificationsOff while notifications stay off (the user
// can turn them on in the system's settings for the app). To ask at a
// better moment, call RequestNotificationPermission. On iOS and macOS
// notifications show as banners while the app is in front too (the
// system hides them otherwise); macOS needs an app bundle (a giode
// build, not go run).
//
// It returns once the system took the notification, or after a moment
// while the permission prompt is still up.
func (a *App) Notify(title, body string) error {
	return a.notify(title, body)
}

// RequestNotificationPermission asks the user to allow the app's
// notifications, where the platform asks (Android 13 and later, iOS,
// macOS), and reports whether they're allowed. The system shows its
// prompt only while the user hasn't decided; later calls just report.
// It blocks until the user answers: call it from a goroutine. Elsewhere
// it reports true.
func (a *App) RequestNotificationPermission(ctx context.Context) (bool, error) {
	return a.requestNotificationPermission(ctx)
}
