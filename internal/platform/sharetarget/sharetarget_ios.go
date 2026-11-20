//go:build ios

package sharetarget

/*
#cgo CFLAGS: -Werror -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework UIKit

#include <stdlib.h>

char *giodeShareInbox(void);
char *giodeShareScheme(void);
void giodeWatchShares(void);
*/
import "C"

import (
	"strings"
	"sync"
	"unsafe"

	"github.com/xmhrnx/giode/internal/mobile"
)

var listener struct {
	sync.Mutex
	deliver func(mobile.Shared)
	inbox   string
}

// goStringFree converts and frees a malloc'ed C string; NULL is "".
func goStringFree(s *C.char) string {
	if s == nil {
		return ""
	}
	defer C.free(unsafe.Pointer(s))
	return C.GoString(s)
}

// Listen picks up what the share extension (app.share_target) left in
// the App Group's inbox, each time the app becomes active.
func Listen(deliver func(mobile.Shared)) error {
	inbox := goStringFree(C.giodeShareInbox())
	if inbox == "" {
		return nil
	}
	listener.Lock()
	listener.deliver, listener.inbox = deliver, inbox
	listener.Unlock()
	C.giodeWatchShares()
	go collect(inbox, deliver)
	return nil
}

//export giodeOnSharesMaybe
func giodeOnSharesMaybe() {
	listener.Lock()
	deliver, inbox := listener.deliver, listener.inbox
	listener.Unlock()
	if deliver != nil {
		go collect(inbox, deliver)
	}
}

// IsShareScheme reports whether a URL scheme is the one the share
// extension opens the app with, which isn't for OnOpenURL.
func IsShareScheme(scheme string) bool {
	s := goStringFree(C.giodeShareScheme())
	return s != "" && strings.EqualFold(s, scheme)
}
