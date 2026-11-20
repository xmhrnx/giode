package keepawake

import (
	"context"
	"fmt"
	"sync"

	"gioui.org/app"

	"github.com/xmhrnx/giode/internal/mobile"
	"github.com/xmhrnx/giode/internal/platform"
)

// androidHaptic maps giode.HapticKind to
// android.view.HapticFeedbackConstants.
var androidHaptic = [...]int{
	3,  // HapticLight: KEYBOARD_TAP
	1,  // HapticMedium: VIRTUAL_KEY
	0,  // HapticHeavy: LONG_PRESS
	16, // HapticSuccess: CONFIRM
	17, // HapticWarning: REJECT
}

// Haptic plays the view's haptic feedback.
func Haptic(h platform.Host, kind int) {
	if kind < 0 || kind >= len(androidHaptic) {
		return
	}
	platform.OnAndroidMain(context.Background(), h, func(view uintptr) error {
		return mobile.Haptic(app.JavaVM(), app.AppContext(), view, androidHaptic[kind])
	})
}

// KeepScreenOn sets the window's FLAG_KEEP_SCREEN_ON.
func KeepScreenOn(ctx context.Context, h platform.Host, on bool) error {
	return platform.OnAndroidMain(ctx, h, func(view uintptr) error {
		return mobile.KeepScreenOn(app.JavaVM(), app.AppContext(), view, on)
	})
}

// keepAlives counts KeepAlive holds; the service runs while any is
// held, showing the latest reason.
var keepAlives struct {
	sync.Mutex
	n int
}

// KeepAlive starts giode's foreground service. onStart runs first, once
// the service is known to be declared: the service's notification needs
// the notification permission, without which Android hides it (the
// service still runs).
func KeepAlive(h platform.Host, reason string, onStart func()) (func(), error) {
	vm, ctx := app.JavaVM(), app.AppContext()
	ok, err := mobile.KeepAliveDeclared(vm, ctx)
	if err != nil {
		return func() {}, err
	}
	if !ok {
		return func() {}, fmt.Errorf("%w: the app manifest doesn't declare giode's foreground service (see docs/platform.md)", platform.ErrNotSupported)
	}
	if onStart != nil {
		onStart()
	}
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
