package picker

/*
#cgo CFLAGS: -Werror -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework UIKit -framework PhotosUI

#include <CoreFoundation/CoreFoundation.h>

int giodePickMedia(CFTypeRef vc, int limit, int photos, int videos);
*/
import "C"

import (
	"context"
	"errors"
	"io"
	"os"
	"unsafe"

	"github.com/xmhrnx/giode/internal/platform"
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

// PickMedia shows PHPicker, which copies the originals into the app's
// temporary directory, and opens them. Before iOS 14 it returns
// ErrUseFallback.
func PickMedia(ctx context.Context, h platform.Host, limit int, photos, videos bool) ([]io.ReadCloser, error) {
	return await(ctx, func() ([]io.ReadCloser, error) {
		dialogMu.Lock()
		defer dialogMu.Unlock()
		vc, err := viewController(h)
		if err != nil {
			return nil, err
		}
		b := func(v bool) C.int {
			if v {
				return 1
			}
			return 0
		}
		if C.giodePickMedia(vc, C.int(max(limit, 0)), b(photos), b(videos)) == 0 {
			return nil, ErrUseFallback // PHPicker needs iOS 14
		}
		r := <-mediaPicked
		if len(r.paths) == 0 {
			if r.err != nil {
				return nil, r.err
			}
			return nil, platform.ErrCanceled
		}
		files := make([]io.ReadCloser, 0, len(r.paths))
		for _, p := range r.paths {
			f, err := os.Open(p)
			if err != nil {
				for _, f := range files {
					f.Close()
				}
				return nil, err
			}
			files = append(files, f)
		}
		return files, nil
	}, func(fs []io.ReadCloser) {
		for _, f := range fs {
			f.Close()
		}
	})
}
