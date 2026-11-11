package giode

import (
	"context"
	"fmt"
	"sync"

	"gioui.org/app"

	"github.com/xmhrnx/giode/internal/mobile"
)

func (a *App) haptic(k HapticKind) {
	a.onAndroidMain(context.Background(), func(view uintptr) error {
		return mobile.Haptic(app.JavaVM(), app.AppContext(), view, androidHaptic[k])
	})
}

func (a *App) keepScreenOn(ctx context.Context, on bool) error {
	return a.onAndroidMain(ctx, func(view uintptr) error {
		return mobile.KeepScreenOn(app.JavaVM(), app.AppContext(), view, on)
	})
}

// keepAlives counts KeepAlive holds; the service runs while any is
// held, showing the latest reason.
var keepAlives struct {
	sync.Mutex
	n int
}

func (a *App) keepAlive(reason string) (func(), error) {
	vm, ctx := app.JavaVM(), app.AppContext()
	ok, err := mobile.KeepAliveDeclared(vm, ctx)
	if err != nil {
		return func() {}, err
	}
	if !ok {
		return func() {}, fmt.Errorf("%w: the app manifest doesn't declare giode's foreground service (see docs/platform.md)", ErrNotSupported)
	}
	// The service's notification needs the notification permission:
	// without it Android hides it (the service still runs).
	a.askNotificationsOnce()
	keepAlives.Lock()
	defer keepAlives.Unlock()
	if err := mobile.StartKeepAlive(vm, ctx, reason); err != nil {
		return func() {}, err
	}
	keepAlives.n++
	var once sync.Once
	return func() {
		once.Do(func() {
			keepAlives.Lock()
			defer keepAlives.Unlock()
			if keepAlives.n--; keepAlives.n == 0 {
				mobile.StopKeepAlive(vm, ctx)
			}
		})
	}, nil
}
