package giode

/*
#include <CoreFoundation/CoreFoundation.h>

int giodePickMedia(CFTypeRef vc, int limit, int photos, int videos);
*/
import "C"

import (
	"context"
	"errors"
	"os"
	"unsafe"
)

type mediaResult struct {
	paths []string
	err   error
}

// mediaPicked receives what giodePickMedia's picker returns: no paths
// when the user picked nothing.
var mediaPicked = make(chan mediaResult, 1)

//export giodeMediaPicked
func giodeMediaPicked(paths *C.char, n C.int, errMsg *C.char) {
	var r mediaResult
	if paths != nil {
		r.paths = splitPaths(C.GoBytes(unsafe.Pointer(paths), n))
	}
	if errMsg != nil {
		r.err = errors.New(C.GoString(errMsg))
	}
	mediaPicked <- r
}

// pickMedia shows PHPicker, which copies the originals into the app's
// temporary directory, and opens them.
func (a *App) pickMedia(ctx context.Context, opts MediaOptions) ([]PickedFile, error) {
	files, err := a.pickMediaPHPicker(ctx, opts)
	if errors.Is(err, errNoPHPicker) {
		return a.pickMediaFiles(ctx, opts)
	}
	return files, err
}

// errNoPHPicker is PHPicker missing, before iOS 14.
var errNoPHPicker = errors.New("PHPicker needs iOS 14")

func (a *App) pickMediaPHPicker(ctx context.Context, opts MediaOptions) ([]PickedFile, error) {
	return await(ctx, func() ([]PickedFile, error) {
		fileDialogMu.Lock()
		defer fileDialogMu.Unlock()
		vc, err := a.viewController()
		if err != nil {
			return nil, err
		}
		b := func(v bool) C.int {
			if v {
				return 1
			}
			return 0
		}
		if C.giodePickMedia(vc, C.int(max(opts.Limit, 0)), b(opts.Photos), b(opts.Videos)) == 0 {
			return nil, errNoPHPicker
		}
		r := <-mediaPicked
		if len(r.paths) == 0 {
			if r.err != nil {
				return nil, r.err
			}
			return nil, ErrCanceled
		}
		files := make([]PickedFile, 0, len(r.paths))
		for _, p := range r.paths {
			f, err := os.Open(p)
			if err != nil {
				for _, f := range files {
					f.Close()
				}
				return nil, err
			}
			files = append(files, describe(f))
		}
		return files, nil
	}, func(fs []PickedFile) {
		for _, f := range fs {
			f.Close()
		}
	})
}
