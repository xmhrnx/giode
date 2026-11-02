package push

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#cgo ios LDFLAGS: -framework UIKit -framework UserNotifications
#cgo !ios LDFLAGS: -framework AppKit -framework UserNotifications

void giodePushRegister(void);
void giodePushUnregister(void);
void giodePushRequestPermission(int id);
void giodePushStart(void);
void giodePushReclaimDelegate(void);
*/
import "C"

import (
	"context"
	"errors"
	"sync"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/internal/bridge"
)

var (
	tokenMu   sync.Mutex
	lastToken string
	waiters   []chan tokenResult

	permissions bridge.Requests[bool]
)

type tokenResult struct {
	token string
	err   error
}

func init() {
	// gioui.org/x/notify, behind App.Notify on macOS, takes over the
	// notification center's delegate; giode/push takes it back, passing
	// local notifications on.
	bridge.AfterNotifier = func() { C.giodePushReclaimDelegate() }
}

func start(a *giode.App) error {
	C.giodePushStart()
	return nil
}

func token(ctx context.Context) (string, error) {
	ch := make(chan tokenResult, 1)
	tokenMu.Lock()
	waiters = append(waiters, ch)
	tokenMu.Unlock()
	C.giodePushRegister()
	select {
	case r := <-ch:
		return r.token, r.err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func requestPermission(ctx context.Context, a *giode.App) (bool, error) {
	id, ch := permissions.New()
	C.giodePushRequestPermission(C.int(id))
	return permissions.Wait(ctx, id, ch)
}

func unregister() error {
	C.giodePushUnregister()
	tokenMu.Lock()
	lastToken = ""
	tokenMu.Unlock()
	return nil
}

//export giodePushToken
func giodePushToken(tok, errMsg *C.char) {
	r := tokenResult{token: C.GoString(tok)}
	if errMsg != nil {
		r.err = errors.New("push: " + C.GoString(errMsg))
	}
	tokenMu.Lock()
	ws := waiters
	waiters = nil
	changed := r.err == nil && r.token != lastToken
	if changed {
		lastToken = r.token
	}
	tokenMu.Unlock()
	for _, w := range ws {
		w <- r
	}
	if changed {
		deliver(event{kind: evToken, token: r.token})
	}
}

//export giodePushMessage
func giodePushMessage(js *C.char, opened C.int) {
	received(C.GoString(js), opened != 0)
}

//export giodePushPermission
func giodePushPermission(id, granted C.int) {
	permissions.Done(int(id), granted != 0)
}
