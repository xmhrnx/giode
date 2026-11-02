package giode

import (
	"sync"

	"gioui.org/x/notify"

	"github.com/xmhrnx/giode/internal/bridge"
)

var (
	notifierOnce sync.Once
	notifier     notify.Notifier
	notifierErr  error
)

// Notify shows a system notification with title and body - to tell
// the user a run or a download finished while the app was in the
// background. It works on Android, macOS, Windows and Linux (through
// gioui.org/x/notify). On Android 13 and later the system asks the
// user for permission the first time; an app built with a target SDK
// of 33 or more can't show notifications at all, since gogio can't
// declare the POST_NOTIFICATIONS permission.
func (a *App) Notify(title, body string) error {
	notifierOnce.Do(func() {
		notifier, notifierErr = notify.NewNotifier()
		if bridge.AfterNotifier != nil {
			bridge.AfterNotifier()
		}
	})
	if notifierErr != nil {
		return notifierErr
	}
	_, err := notifier.CreateNotification(title, body)
	return err
}
