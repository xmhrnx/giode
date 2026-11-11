package giode

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework Foundation -framework UserNotifications

#include <stdlib.h>

int giodeNotifyAvailable(void);
void giodeNotifyInstall(void);
void giodeNotify(int id, const char *title, const char *body);
*/
import "C"

import (
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

func (a *App) notify(title, body string) error {
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
func giodeNotified(id C.int, errMsg *C.char) {
	notifyMu.Lock()
	ch := notifyWait[int(id)]
	delete(notifyWait, int(id))
	notifyMu.Unlock()
	if ch == nil {
		return
	}
	var err error
	if errMsg != nil {
		err = errors.New(C.GoString(errMsg))
	}
	ch <- err
}
