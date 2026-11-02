package biometric

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework Foundation -framework LocalAuthentication

#include <stdlib.h>

int giodeBiometryKind(void);
void giodeBiometricAuthenticate(int id, const char *reason, int passcode);
*/
import "C"

import (
	"context"
	"unsafe"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/internal/bridge"
)

const supported = true

var requests bridge.Requests[error]

func available() Kind { return Kind(C.giodeBiometryKind()) }

func authenticate(ctx context.Context, a *giode.App, opts Options) error {
	if opts.Reason == "" {
		opts.Reason = "Unlock " + opts.Title
	}
	id, ch := requests.New()
	r := C.CString(opts.Reason)
	defer C.free(unsafe.Pointer(r))
	p := 0
	if opts.AllowPasscode {
		p = 1
	}
	C.giodeBiometricAuthenticate(C.int(id), r, C.int(p))
	err, werr := requests.Wait(ctx, id, ch)
	if werr != nil {
		return werr
	}
	return err
}

//export giodeBiometricDone
func giodeBiometricDone(id, code C.int, msg *C.char) {
	m := ""
	if msg != nil {
		m = C.GoString(msg)
	}
	requests.Done(int(id), resultError(int(code), m))
}
