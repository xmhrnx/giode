package giode

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework Foundation -framework UserNotifications

#include <stdlib.h>

int giodeNotifyAvailable(void);
void giodeNotifyInstall(void);
void giodeNotify(int id, const char *title, const char *body);
void giodeNotifyRequest(int id);
*/
import "C"

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
	"unsafe"

	"github.com/xmhrnx/giode/internal/bridge"
)

// Notifications on iOS and macOS go through UNUserNotificationCenter,
// with a delegate that shows them while the app is in front.

var (
	notifyOnce sync.Once
	notifyOK   bool
	notifyMu   sync.Mutex
	notifyNext int
	notifyWait = map[int]chan error{}
)

// notifySetup installs the notification center's delegate, once, and
// reports whether notifications can work: they need an app bundle.
func notifySetup() error {
	notifyOnce.Do(func() {
		notifyOK = C.giodeNotifyAvailable() != 0
		if notifyOK {
			C.giodeNotifyInstall()
			if bridge.AfterNotifier != nil {
				bridge.AfterNotifier()
			}
		}
	})
	if !notifyOK {
		return fmt.Errorf("%w: notifications need an app bundle (build the app with giode)", ErrNotSupported)
	}
	return nil
}

func (a *App) notify(title, body string) error {
	if err := notifySetup(); err != nil {
		return err
	}
	ch := make(chan error, 1)
	notifyMu.Lock()
	notifyNext++
	id := notifyNext
	notifyWait[id] = ch
	notifyMu.Unlock()
	ct, cb := C.CString(title), C.CString(body)
	C.giodeNotify(C.int(id), ct, cb)
	C.free(unsafe.Pointer(ct))
	C.free(unsafe.Pointer(cb))
	if onMainThread() {
		return nil // never hold up the main thread
	}
	select {
	case err := <-ch:
		return err
	case <-time.After(2 * time.Second):
		// The permission prompt is up; the notification follows if the
		// user allows it.
		return nil
	}
}

//export giodeNotified
func giodeNotified(id C.int, denied C.int, errMsg *C.char) {
	notifyMu.Lock()
	ch := notifyWait[int(id)]
	delete(notifyWait, int(id))
	notifyMu.Unlock()
	if ch == nil {
		return
	}
	var err error
	switch {
	case denied != 0 && errMsg != nil:
		err = fmt.Errorf("%w: %s", ErrNotificationsOff, C.GoString(errMsg))
	case denied != 0:
		err = ErrNotificationsOff
	case errMsg != nil:
		err = errors.New(C.GoString(errMsg))
	}
	ch <- err
}

var allowWait = map[int]chan bool{}

func (a *App) requestNotificationPermission(ctx context.Context) (bool, error) {
	if err := notifySetup(); err != nil {
		return false, err
	}
	ch := make(chan bool, 1)
	notifyMu.Lock()
	notifyNext++
	id := notifyNext
	allowWait[id] = ch
	notifyMu.Unlock()
	C.giodeNotifyRequest(C.int(id))
	select {
	case ok := <-ch:
		return ok, nil
	case <-ctx.Done():
		notifyMu.Lock()
		delete(allowWait, id)
		notifyMu.Unlock()
		return false, ctx.Err()
	}
}

//export giodeNotifyAllowed
func giodeNotifyAllowed(id C.int, ok C.int) {
	notifyMu.Lock()
	ch := allowWait[int(id)]
	delete(allowWait, int(id))
	notifyMu.Unlock()
	if ch != nil {
		ch <- ok != 0
	}
}
