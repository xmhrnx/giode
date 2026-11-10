package giode

/*
#cgo CFLAGS: -Werror -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework UIKit -framework PhotosUI -framework Photos -framework AVFoundation

#include <stdint.h>
#include <stdlib.h>
#include <CoreFoundation/CoreFoundation.h>

void giodeOpenURL(const char *url);
void giodeShare(CFTypeRef vc, const char *text, const char *path, const char *subject);
void giodeHaptic(int kind);
void giodeIdleTimerDisabled(int disabled);
uintptr_t giodeBeginBackgroundTask(const char *name);
void giodeEndBackgroundTask(uintptr_t task);
void giodeWatchLifecycle(void);
int giodePickPhoto(CFTypeRef vc, int id, int maxSide);
int giodeHasInfoKey(const char *key);
int giodePickCamera(CFTypeRef vc, int id, int maxSide);
int giodePickPhotos(CFTypeRef vc, int id, int maxSide, int limit);
void giodeShareFiles(CFTypeRef vc, const char *paths);
void giodeSaveToPhotos(int id, const char *path, int video);
*/
import "C"

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"unsafe"
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

func (a *App) openURL(ctx context.Context, u string) error {
	cu, free := cstr(u)
	defer free()
	C.giodeOpenURL(cu)
	return nil
}

func (t ShareText) share(ctx context.Context, a *App) error {
	vc, err := a.viewController()
	if err != nil {
		return err
	}
	ct, f1 := cstr(t.Text)
	defer f1()
	cs, f2 := cstr(t.Subject)
	defer f2()
	C.giodeShare(vc, ct, nil, cs)
	return nil
}

func (f ShareFile) share(ctx context.Context, a *App) error {
	vc, err := a.viewController()
	if err != nil {
		return err
	}
	cp, free := cstr(f.Path)
	defer free()
	C.giodeShare(vc, nil, cp, nil)
	return nil
}

func (f ShareFiles) share(ctx context.Context, a *App) error {
	if len(f.Paths) == 0 {
		return nil
	}
	vc, err := a.viewController()
	if err != nil {
		return err
	}
	cp, free := cstr(strings.Join(f.Paths, "\n"))
	defer free()
	C.giodeShareFiles(vc, cp)
	return nil
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

var savedToPhotos struct {
	sync.Mutex
	next int
	m    map[int]chan error
}

func (a *App) saveToPhotosIn(ctx context.Context, path, mime, album string) error {
	if err := requireInfoKeys("NSPhotoLibraryAddUsageDescription"); err != nil {
		return err
	}
	savedToPhotos.Lock()
	if savedToPhotos.m == nil {
		savedToPhotos.m = map[int]chan error{}
	}
	savedToPhotos.next++
	id := savedToPhotos.next
	ch := make(chan error, 1)
	savedToPhotos.m[id] = ch
	savedToPhotos.Unlock()
	cp, free := cstr(path)
	defer free()
	video := 0
	if strings.HasPrefix(mime, "video/") {
		video = 1
	}
	C.giodeSaveToPhotos(C.int(id), cp, C.int(video))
	select {
	case err := <-ch:
		return err
	case <-ctx.Done():
		savedToPhotos.Lock()
		delete(savedToPhotos.m, id)
		savedToPhotos.Unlock()
		return ctx.Err()
	}
}

//export giodeSavedToPhotos
func giodeSavedToPhotos(id C.int, errMsg *C.char) {
	savedToPhotos.Lock()
	ch := savedToPhotos.m[int(id)]
	delete(savedToPhotos.m, int(id))
	savedToPhotos.Unlock()
	if ch == nil {
		return
	}
	var err error
	if errMsg != nil {
		err = errors.New(C.GoString(errMsg))
	}
	ch <- err
}

var lifecycleApp struct {
	sync.Mutex
	app *App
}

// watchLifecycle reports UIApplication's background and foreground
// notifications to OnBackground and OnForeground.
func (a *App) watchLifecycle() {
	if a.cfg.OnBackground == nil && a.cfg.OnForeground == nil {
		return
	}
	lifecycleApp.Lock()
	lifecycleApp.app = a
	lifecycleApp.Unlock()
	C.giodeWatchLifecycle()
}

//export giodeOnBackground
func giodeOnBackground() {
	lifecycleApp.Lock()
	a := lifecycleApp.app
	lifecycleApp.Unlock()
	if a != nil && a.cfg.OnBackground != nil {
		a.cfg.OnBackground()
	}
}

//export giodeOnForeground
func giodeOnForeground() {
	lifecycleApp.Lock()
	a := lifecycleApp.app
	lifecycleApp.Unlock()
	if a != nil && a.cfg.OnForeground != nil {
		a.cfg.OnForeground()
	}
}

// iOS reports going to the background through UIApplication, not the
// window state.
const lifecycleFromWindowState = false

func (a *App) haptic(k HapticKind) {
	C.giodeHaptic(C.int(k))
}

func (a *App) keepScreenOn(ctx context.Context, on bool) error {
	d := C.int(0)
	if on {
		d = 1
	}
	C.giodeIdleTimerDisabled(d)
	return nil
}

func (a *App) keepAlive(reason string) (func(), error) {
	cr, free := cstr(reason)
	defer free()
	task := C.giodeBeginBackgroundTask(cr)
	var once sync.Once
	return func() { once.Do(func() { C.giodeEndBackgroundTask(task) }) }, nil
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
	if opts.Source == Camera {
		// Using the camera without its usage text crashes the app.
		if err := requireInfoKeys("NSCameraUsageDescription"); err != nil {
			return PickedFile{}, err
		}
	}
	vc, err := a.viewController()
	if err != nil {
		return PickedFile{}, err
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
	if opts.Source == Camera {
		switch C.giodePickCamera(vc, C.int(id), C.int(opts.MaxSide)) {
		case 0:
			forget()
			return PickedFile{}, fmt.Errorf("%w: this device has no camera", ErrNotSupported)
		case -1:
			forget()
			return PickedFile{}, errors.New("camera access isn't allowed for this app (Settings > Privacy & Security > Camera)")
		}
	} else if C.giodePickPhoto(vc, C.int(id), C.int(opts.MaxSide)) == 0 {
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
