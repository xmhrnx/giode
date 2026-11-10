package gallery

/*
#cgo CFLAGS: -Werror -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework Foundation -framework Photos

#include <stdlib.h>

void giodeGallerySave(int id, const char *path, int video);
int giodeGalleryHasInfoKey(const char *key);
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
	optin.SavePhoto = save
}

var reqs struct {
	sync.Mutex
	next int
	ch   map[int]chan error
}

func save(ctx context.Context, path string, video bool) error {
	k := C.CString("NSPhotoLibraryAddUsageDescription")
	ok := C.giodeGalleryHasInfoKey(k) != 0
	C.free(unsafe.Pointer(k))
	if !ok {
		return fmt.Errorf("%w: Info.plist lacks NSPhotoLibraryAddUsageDescription (ios.usage.photos_add in giode.toml)", giode.ErrNotSupported)
	}
	reqs.Lock()
	if reqs.ch == nil {
		reqs.ch = map[int]chan error{}
	}
	reqs.next++
	id := reqs.next
	ch := make(chan error, 1)
	reqs.ch[id] = ch
	reqs.Unlock()
	cp := C.CString(path)
	defer C.free(unsafe.Pointer(cp))
	v := C.int(0)
	if video {
		v = 1
	}
	C.giodeGallerySave(C.int(id), cp, v)
	select {
	case err := <-ch:
		return err
	case <-ctx.Done():
		reqs.Lock()
		delete(reqs.ch, id)
		reqs.Unlock()
		return ctx.Err()
	}
}

//export giodeGallerySaved
func giodeGallerySaved(id C.int, errMsg *C.char) {
	reqs.Lock()
	ch := reqs.ch[int(id)]
	delete(reqs.ch, int(id))
	reqs.Unlock()
	if ch == nil {
		return
	}
	var err error
	if errMsg != nil {
		err = errors.New(C.GoString(errMsg))
	}
	ch <- err
}
