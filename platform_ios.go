package giode

/*
#cgo CFLAGS: -Werror -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework UIKit -framework PhotosUI -framework AVFoundation

#include <stdint.h>
#include <stdlib.h>
#include <CoreFoundation/CoreFoundation.h>

int giodePickPhoto(CFTypeRef vc, int id, int maxSide);
int giodeHasInfoKey(const char *key);
int giodePickPhotos(CFTypeRef vc, int id, int maxSide, int limit);
*/
import "C"

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"unsafe"

	"github.com/xmhrnx/giode/internal/optin"
)

func cstr(s string) (*C.char, func()) {
	c := C.CString(s)
	return c, func() { C.free(unsafe.Pointer(c)) }
}

// viewController returns the UIViewController behind the window.
func (a *App) viewController() (C.CFTypeRef, error) {
	h := a.NativeHandle()
	if h == 0 {
		return 0, ErrNotSupported
	}
	return C.CFTypeRef(h), nil
}

// photoSets collects multiple picks.
var photoSets struct {
	sync.Mutex
	next int
	m    map[int]*photoSet
}

type photoSet struct {
	files []PickedFile
	done  chan error
}

func (a *App) pickPhotos(ctx context.Context, opts PhotoOptions, limit int) ([]PickedFile, error) {
	vc, err := a.viewController()
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
	if C.giodePickPhotos(vc, C.int(id), C.int(opts.MaxSide), C.int(limit)) == 0 {
		forget()
		return nil, ErrNotSupported // PHPicker needs iOS 14
	}
	select {
	case err := <-set.done:
		if err != nil {
			return nil, err
		}
		if len(set.files) == 0 {
			return nil, ErrCanceled
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
		set.files = append(set.files, jpegFile(C.GoString(name), C.GoBytes(data, n)))
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

func (a *App) saveToPhotosIn(ctx context.Context, path, mime, album string) error {
	if optin.SavePhoto == nil {
		return notLinked("gallery")
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

func (a *App) pickPhoto(ctx context.Context, opts PhotoOptions) (PickedFile, error) {
	vc, err := a.viewController()
	if err != nil {
		return PickedFile{}, err
	}
	if opts.Source == Camera {
		if optin.TakePhoto == nil {
			return PickedFile{}, notLinked("takephoto")
		}
		jpeg, err := optin.TakePhoto(ctx, uintptr(vc), opts.MaxSide)
		if err != nil {
			return PickedFile{}, err
		}
		return jpegFile("photo", jpeg), nil
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
	if C.giodePickPhoto(vc, C.int(id), C.int(opts.MaxSide)) == 0 {
		forget()
		return PickedFile{}, ErrNotSupported // PHPicker needs iOS 14
	}
	select {
	case r := <-ch:
		if r.err != nil {
			return PickedFile{}, r.err
		}
		return jpegFile(r.name, r.jpeg), nil
	case <-ctx.Done():
		return PickedFile{}, ctx.Err()
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
		r.err = ErrCanceled
	default:
		r.jpeg = C.GoBytes(data, n)
		r.name = C.GoString(name)
	}
	ch <- r
}

// requireInfoKeys returns an error wrapping ErrNotSupported unless the
// app's Info.plist has every key: iOS ends an app that uses the camera
// or microphone without the text explaining why.
func requireInfoKeys(keys ...string) error {
	var missing []string
	for _, k := range keys {
		ck, free := cstr(k)
		ok := C.giodeHasInfoKey(ck) != 0
		free()
		if !ok {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("%w: Info.plist lacks %s (see docs/platform.md)", ErrNotSupported, strings.Join(missing, ", "))
	}
	return nil
}

