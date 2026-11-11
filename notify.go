package giode

// Notify shows a system notification with title and body - to tell
// the user a transfer arrived or a download finished. It works on
// Android, iOS, macOS, Windows and Linux.
//
// On iOS and macOS the system asks the user for permission the first
// time, and notifications show as banners while the app is in front
// too (the system hides them otherwise). macOS needs an app bundle (a
// giode build, not go run). On Android 13 and later the system asks
// for permission the first time; the user can turn notifications on
// later in the system's settings for the app.
//
// It returns once the system took the notification, or after a moment
// while the permission prompt is still up.
func (a *App) Notify(title, body string) error {
	return a.notify(title, body)
}
