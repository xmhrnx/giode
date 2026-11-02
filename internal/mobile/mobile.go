// Package mobile reaches the Android services Gio doesn't expose - the
// system share sheet and the activity lifecycle - through a small Java
// helper (mobile_android.java, shipped as mobile_android.jar, which
// gogio packs into the APK). Elsewhere its functions report
// ErrUnsupported.
//
// It takes the JVM, Android context and view as plain handles rather
// than importing gioui.org/app, so it type-checks without the NDK.
package mobile

import (
	"errors"
	"sync/atomic"
)

// ErrUnsupported is returned on platforms without the service.
var ErrUnsupported = errors.New("not supported on this platform")

var onBackground, onForeground atomic.Pointer[func()]

// SetLifecycleHandlers sets the functions run when the app goes to the
// background and comes back; nil clears one. They run on the
// platform's main thread.
func SetLifecycleHandlers(background, foreground func()) {
	store := func(p *atomic.Pointer[func()], fn func()) {
		if fn == nil {
			p.Store(nil)
			return
		}
		p.Store(&fn)
	}
	store(&onBackground, background)
	store(&onForeground, foreground)
}

func notify(p *atomic.Pointer[func()]) {
	if fn := p.Load(); fn != nil {
		(*fn)()
	}
}

// ErrCanceled is returned when the user dismisses a picker.
var ErrCanceled = errors.New("canceled")

// Photo is the result of PickPhoto: a JPEG, its display name, or an
// error (ErrCanceled when the user backed out).
type Photo struct {
	JPEG []byte
	Name string
	Err  error
}

// Photos is the result of PickPhotos: the picked photos, or an error
// (ErrCanceled when the user picked none).
type Photos struct {
	Photos []Photo
	Err    error
}

// Speech is one dictation result: partial while Done is false, final
// when it's true.
type Speech struct {
	Text string
	Done bool
	Err  error
}
