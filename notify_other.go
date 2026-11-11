//go:build !darwin && !android

package giode

import "context"

func (a *App) notify(title, body string) error {
	return a.xnotify(title, body)
}

// requestNotificationPermission reports true: desktops other than
// macOS don't ask.
func (a *App) requestNotificationPermission(ctx context.Context) (bool, error) {
	return true, nil
}
