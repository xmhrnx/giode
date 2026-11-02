package printing

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#cgo ios LDFLAGS: -framework UIKit
#cgo !ios LDFLAGS: -framework AppKit -framework PDFKit

#include <stdlib.h>
#include <CoreFoundation/CoreFoundation.h>

void giodePrint(int id, CFTypeRef view, const char *path, const char *job, int pdf);
*/
import "C"

import (
	"context"
	"errors"
	"unsafe"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/internal/bridge"
)

const supported = true

var requests bridge.Requests[error]

func printFile(ctx context.Context, a *giode.App, path string, opts Options) error {
	view := a.NativeHandle()
	if view == 0 {
		return errors.New("printing: the window isn't open yet")
	}
	id, ch := requests.New()
	cp := C.CString(path)
	defer C.free(unsafe.Pointer(cp))
	cj := C.CString(opts.JobName)
	defer C.free(unsafe.Pointer(cj))
	pdf := 0
	if isPDF(path) {
		pdf = 1
	}
	C.giodePrint(C.int(id), C.CFTypeRef(view), cp, cj, C.int(pdf))
	err, werr := requests.Wait(ctx, id, ch)
	if werr != nil {
		return werr
	}
	return err
}

//export giodePrintDone
func giodePrintDone(id C.int, msg *C.char) {
	var err error
	if msg != nil {
		err = errors.New("printing: " + C.GoString(msg))
	}
	requests.Done(int(id), err)
}
