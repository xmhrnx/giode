//go:build !darwin

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

// xnotify shows a notification through gioui.org/x/notify.
func (a *App) xnotify(title, body string) error {
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
