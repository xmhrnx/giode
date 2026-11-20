package picker

/*
#cgo CFLAGS: -Werror -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework UIKit -framework PhotosUI

#include <stdlib.h>
#include <CoreFoundation/CoreFoundation.h>

int giodePickPhoto(CFTypeRef vc, int id, int maxSide);
int giodePickPhotos(CFTypeRef vc, int id, int maxSide, int limit);
*/
import "C"
import (
	"context"
	"errors"
	"strings"
	"sync"
	"unsafe"

	"github.com/xmhrnx/giode/internal/optin"
	"github.com/xmhrnx/giode/internal/platform"
)

// photoSets collects multiple picks.
var photoSets struct {
	sync.Mutex
	next int
	m    map[int]*photoSet
}

type photoSet struct {
	files []Photo
	done  chan error
}

// PickPhotos shows PHPicker for several photos.
func PickPhotos(ctx context.Context, h platform.Host, maxSide, limit int) ([]Photo, error) {
	vc, err := viewController(h)
	if err != nil {
		return nil, err
	}
	photoSets.Lock()
	if photoSets.m == nil {
		photoSets.m = map[int]*photoSet{}
	}
	photoSets.next++
	id := photoSets.next
	set := &photoSet{done: make(chan error, 1)}
	photoSets.m[id] = set
	photoSets.Unlock()
	forget := func() {
		photoSets.Lock()
		delete(photoSets.m, id)
		photoSets.Unlock()
	}
	if C.giodePickPhotos(vc, C.int(id), C.int(maxSide), C.int(limit)) == 0 {
		forget()
		return nil, platform.ErrNotSupported // PHPicker needs iOS 14
	}
	select {
	case err := <-set.done:
		if err != nil {
			return nil, err
		}
		if len(set.files) == 0 {
			return nil, platform.ErrCanceled
		}
		return set.files, nil
	case <-ctx.Done():
		forget()
		return nil, ctx.Err()
	}
}

//export giodePhotosItem
func giodePhotosItem(id C.int, data unsafe.Pointer, n C.int, name *C.char) {
	photoSets.Lock()
	set := photoSets.m[int(id)]
	photoSets.Unlock()
	if set != nil {
		set.files = append(set.files, Photo{Name: C.GoString(name), JPEG: C.GoBytes(data, n)})
	}
}

//export giodePhotosDone
func giodePhotosDone(id C.int, count C.int, errMsg *C.char) {
	photoSets.Lock()
	set := photoSets.m[int(id)]
	delete(photoSets.m, int(id))
	photoSets.Unlock()
	if set == nil {
		return
	}
	var err error
	if errMsg != nil {
		err = errors.New(C.GoString(errMsg))
	}
	set.done <- err
}

// SaveToPhotos adds the file to the photo library through the gallery
// package.
func SaveToPhotos(ctx context.Context, path, mime, album string) error {
	if optin.SavePhoto == nil {
		return platform.NotLinked("gallery")
	}
	return optin.SavePhoto(ctx, path, strings.HasPrefix(mime, "video/"))
}

var photoReqs struct {
	sync.Mutex
	next int
	ch   map[int]chan photoResult
}

type photoResult struct {
	jpeg []byte
	name string
	err  error
}

// PickPhoto shows PHPicker, or the camera through the takephoto
// package.
func PickPhoto(ctx context.Context, h platform.Host, maxSide int, camera bool) (Photo, error) {
	vc, err := viewController(h)
	if err != nil {
		return Photo{}, err
	}
	if camera {
		if optin.TakePhoto == nil {
			return Photo{}, platform.NotLinked("takephoto")
		}
		jpeg, err := optin.TakePhoto(ctx, uintptr(vc), maxSide)
		if err != nil {
			return Photo{}, err
		}
		return Photo{Name: "photo", JPEG: jpeg}, nil
	}
	photoReqs.Lock()
	if photoReqs.ch == nil {
		photoReqs.ch = map[int]chan photoResult{}
	}
	photoReqs.next++
	id := photoReqs.next
	ch := make(chan photoResult, 1)
	photoReqs.ch[id] = ch
	photoReqs.Unlock()
	forget := func() {
		photoReqs.Lock()
		delete(photoReqs.ch, id)
		photoReqs.Unlock()
	}
	if C.giodePickPhoto(vc, C.int(id), C.int(maxSide)) == 0 {
		forget()
		return Photo{}, platform.ErrNotSupported // PHPicker needs iOS 14
	}
	select {
	case r := <-ch:
		if r.err != nil {
			return Photo{}, r.err
		}
		return Photo{Name: r.name, JPEG: r.jpeg}, nil
	case <-ctx.Done():
		return Photo{}, ctx.Err()
	}
}

//export giodePhotoDone
func giodePhotoDone(id C.int, data unsafe.Pointer, n C.int, name, errMsg *C.char) {
	photoReqs.Lock()
	ch := photoReqs.ch[int(id)]
	delete(photoReqs.ch, int(id))
	photoReqs.Unlock()
	if ch == nil {
		return
	}
	var r photoResult
	switch {
	case errMsg != nil && C.GoString(errMsg) != "":
		r.err = errors.New(C.GoString(errMsg))
	case data == nil:
		r.err = platform.ErrCanceled
	default:
		r.jpeg = C.GoBytes(data, n)
		r.name = C.GoString(name)
	}
	ch <- r
}
