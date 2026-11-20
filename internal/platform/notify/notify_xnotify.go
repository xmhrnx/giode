//go:build !darwin

package notify

import (
	"sync"

	gionotify "gioui.org/x/notify"

	"github.com/xmhrnx/giode/internal/bridge"
)

var (
	notifierOnce sync.Once
	notifier     gionotify.Notifier
	notifierErr  error
)

// xnotify shows a notification through gioui.org/x/notify.
func xnotify(title, body string) error {
	notifierOnce.Do(func() {
		notifier, notifierErr = gionotify.NewNotifier()
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
