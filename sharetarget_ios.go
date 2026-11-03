//go:build ios

package giode

/*
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
)

var shareApp struct {
	sync.Mutex
	app   *App
	inbox string
}

// goStringFree converts and frees a malloc'ed C string; NULL is "".
func goStringFree(s *C.char) string {
	if s == nil {
		return ""
	}
	defer C.free(unsafe.Pointer(s))
	return C.GoString(s)
}

// listenShares picks up what the share extension (app.share_target)
// left in the App Group's inbox, each time the app becomes active.
func (a *App) listenShares() {
	inbox := goStringFree(C.giodeShareInbox())
	if inbox == "" {
		return
	}
	shareApp.Lock()
	shareApp.app, shareApp.inbox = a, inbox
	shareApp.Unlock()
	C.giodeWatchShares()
	go a.collectShares(inbox)
}

//export giodeOnSharesMaybe
func giodeOnSharesMaybe() {
	shareApp.Lock()
	a, inbox := shareApp.app, shareApp.inbox
	shareApp.Unlock()
	if a != nil {
		go a.collectShares(inbox)
	}
}

func init() {
	isShareScheme = func(scheme string) bool {
		s := goStringFree(C.giodeShareScheme())
		return s != "" && strings.EqualFold(s, scheme)
	}
}
