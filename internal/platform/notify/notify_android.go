package notify

import (
	"context"
	"sync/atomic"

	"gioui.org/app"

	"github.com/xmhrnx/giode/internal/mobile"
	"github.com/xmhrnx/giode/internal/platform"
)

// notifyAsked is set once Notify or KeepAlive asked for the
// notification permission in this run.
var notifyAsked atomic.Bool

// Show shows the notification when the app may, and otherwise asks
// for the permission the first time (posting this notification if the
// user allows it); later it returns platform.ErrNotificationsOff.
func Show(h platform.Host, title, body string) error {
	vm, ctx := app.JavaVM(), app.AppContext()
	if on, err := mobile.NotificationsEnabled(vm, ctx); err == nil && !on {
		if notifyAsked.Swap(true) {
			return platform.ErrNotificationsOff
		}
		go func() {
			if ok, _ := RequestPermission(context.Background(), h); ok {
				xnotify(title, body)
			}
		}()
		return nil
	}
	return xnotify(title, body)
}

// AskOnce asks for the notification permission in the background, once
// a run, when notifications are off: KeepAlive's notification needs it
// too.
func AskOnce(h platform.Host) {
	if on, err := mobile.NotificationsEnabled(app.JavaVM(), app.AppContext()); err != nil || on || notifyAsked.Swap(true) {
		return
	}
	go RequestPermission(context.Background(), h)
}

func RequestPermission(ctx context.Context, h platform.Host) (bool, error) {
	var ch <-chan bool
	err := platform.OnAndroidMain(ctx, h, func(view uintptr) error {
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
