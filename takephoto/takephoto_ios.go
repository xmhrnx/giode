package takephoto

/*
#cgo CFLAGS: -Werror -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework UIKit -framework AVFoundation

#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>

int giodeTakePhoto(CFTypeRef vc, int id, int maxSide);
int giodeTakePhotoHasInfoKey(const char *key);
*/
import "C"

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/internal/optin"
)

func init() {
	optin.TakePhoto = take
}

type result struct {
	jpeg []byte
	err  error
}

var reqs struct {
	sync.Mutex
	next int
	ch   map[int]chan result
}

func take(ctx context.Context, vc uintptr, maxSide int) ([]byte, error) {
	// Using the camera without its usage text crashes the app.
	k := C.CString("NSCameraUsageDescription")
	ok := C.giodeTakePhotoHasInfoKey(k) != 0
	C.free(unsafe.Pointer(k))
	if !ok {
		return nil, fmt.Errorf("%w: Info.plist lacks NSCameraUsageDescription (ios.usage.camera in giode.toml)", giode.ErrNotSupported)
	}
	reqs.Lock()
	if reqs.ch == nil {
		reqs.ch = map[int]chan result{}
	}
	reqs.next++
	id := reqs.next
	ch := make(chan result, 1)
	reqs.ch[id] = ch
	reqs.Unlock()
	forget := func() {
		reqs.Lock()
		delete(reqs.ch, id)
		reqs.Unlock()
	}
	switch C.giodeTakePhoto(C.CFTypeRef(vc), C.int(id), C.int(maxSide)) {
	case 0:
		forget()
		return nil, fmt.Errorf("%w: this device has no camera", giode.ErrNotSupported)
	case -1:
		forget()
		return nil, errors.New("camera access isn't allowed for this app (Settings > Privacy & Security > Camera)")
	}
	select {
	case r := <-ch:
		return r.jpeg, r.err
	case <-ctx.Done():
		forget()
		return nil, ctx.Err()
	}
}

//export giodeTakePhotoDone
func giodeTakePhotoDone(id C.int, data unsafe.Pointer, n C.int, errMsg *C.char) {
	reqs.Lock()
	ch := reqs.ch[int(id)]
	delete(reqs.ch, int(id))
	reqs.Unlock()
	if ch == nil {
		return
	}
	var r result
	switch {
	case errMsg != nil:
		r.err = errors.New(C.GoString(errMsg))
	case data == nil:
		r.err = giode.ErrCanceled
	default:
		r.jpeg = C.GoBytes(data, n)
	}
	ch <- r
}
