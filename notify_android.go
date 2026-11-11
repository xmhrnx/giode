package giode

import (
	"context"
	"sync/atomic"

	"gioui.org/app"

	"github.com/xmhrnx/giode/internal/mobile"
)

// notifyAsked is set once Notify or KeepAlive asked for the
// notification permission in this run.
var notifyAsked atomic.Bool

// notify shows the notification when the app may, and otherwise asks
// for the permission the first time (posting this notification if the
// user allows it); later it returns ErrNotificationsOff.
func (a *App) notify(title, body string) error {
	vm, ctx := app.JavaVM(), app.AppContext()
	if on, err := mobile.NotificationsEnabled(vm, ctx); err == nil && !on {
		if notifyAsked.Swap(true) {
			return ErrNotificationsOff
		}
		go func() {
			if ok, _ := a.RequestNotificationPermission(context.Background()); ok {
				a.xnotify(title, body)
			}
		}()
		return nil
	}
	return a.xnotify(title, body)
}

// askNotificationsOnce asks for the notification permission in the
// background, once a run, when notifications are off: KeepAlive's
// notification needs it too.
func (a *App) askNotificationsOnce() {
	if on, err := mobile.NotificationsEnabled(app.JavaVM(), app.AppContext()); err != nil || on || notifyAsked.Swap(true) {
		return
	}
	go a.RequestNotificationPermission(context.Background())
}

func (a *App) requestNotificationPermission(ctx context.Context) (bool, error) {
	var ch <-chan bool
	err := a.onAndroidMain(ctx, func(view uintptr) error {
		var err error
		ch, err = mobile.RequestNotifications(app.JavaVM(), app.AppContext(), view)
		return err
	})
	if err != nil {
		return false, err
	}
	select {
	case ok := <-ch:
		return ok, nil
	case <-ctx.Done():
		return false, ctx.Err()
	}
}
