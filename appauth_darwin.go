//go:build darwin

package giode

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework AuthenticationServices

#include <stdint.h>
#include <stdlib.h>

void giodeAuthStart(uintptr_t view, int id, char *url, char *scheme, int ephemeral);
void giodeAuthCancel(int id);
*/
import "C"

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"sync"
	"unsafe"

	"github.com/xmhrnx/giode/auth"
)

// authResult is how an ASWebAuthenticationSession ended.
type authResult struct {
	url  string
	kind int // 0 the callback URL, 1 canceled, 2 an error message
}

var authSessions = struct {
	sync.Mutex
	next int
	done map[int]chan authResult
}{done: map[int]chan authResult{}}

//export giodeAuthDone
func giodeAuthDone(id C.int, s *C.char, kind C.int) {
	authSessions.Lock()
	ch := authSessions.done[int(id)]
	delete(authSessions.done, int(id))
	authSessions.Unlock()
	if ch != nil {
		ch <- authResult{C.GoString(s), int(kind)}
	}
}

func (a *App) openSignIn(ctx context.Context, u string) error { return a.OpenURL(ctx, u) }

// authSession runs ASWebAuthenticationSession, the sign-in sheet that
// shares Safari's cookies and passkeys, on authURL until it redirects to
// scheme. https redirects (iOS 17.4 and macOS 14.4 match those too) go
// through the browser and the app's universal link instead.
func (a *App) authSession(ctx context.Context, authURL, scheme string, ephemeral bool) (*url.URL, error) {
	if scheme == "" || strings.EqualFold(scheme, "http") || strings.EqualFold(scheme, "https") {
		return nil, auth.ErrUnsupported
	}
	view := a.authAnchorView()
	ch := make(chan authResult, 1)
	authSessions.Lock()
	authSessions.next++
	id := authSessions.next
	authSessions.done[id] = ch
	authSessions.Unlock()
	cu, cs := C.CString(authURL), C.CString(scheme)
	defer C.free(unsafe.Pointer(cu))
	defer C.free(unsafe.Pointer(cs))
	eph := C.int(0)
	if ephemeral {
		eph = 1
	}
	C.giodeAuthStart(C.uintptr_t(view), C.int(id), cu, cs, eph)
	var r authResult
	select {
	case r = <-ch:
	case <-ctx.Done():
		C.giodeAuthCancel(C.int(id))
		return nil, ctx.Err()
	}
	switch r.kind {
	case 0:
		return url.Parse(r.url)
	case 1:
		return nil, auth.ErrCanceled
	}
	return nil, errors.New("auth: " + r.url)
}
