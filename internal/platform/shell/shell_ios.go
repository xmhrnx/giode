package shell

/*
#cgo CFLAGS: -Werror -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework UIKit

#include <stdlib.h>
#include <CoreFoundation/CoreFoundation.h>

void giodeOpenURL(const char *url);
void giodeShare(CFTypeRef vc, const char *text, const char *path, const char *subject);
void giodeShareFiles(CFTypeRef vc, const char *paths);
void giodeOpenIn(CFTypeRef vc, const char *path);
*/
import "C"

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"github.com/xmhrnx/giode/internal/platform"
	"github.com/xmhrnx/giode/internal/platform/dirs"
)

func cstr(s string) (*C.char, func()) {
	c := C.CString(s)
	return c, func() { C.free(unsafe.Pointer(c)) }
}

// viewController is the UIViewController behind the window.
func viewController(h platform.Host) (C.CFTypeRef, error) {
	v := h.NativeHandle()
	if v == 0 {
		return 0, platform.ErrNotSupported
	}
	return C.CFTypeRef(v), nil
}

// OpenURL opens u with UIApplication openURL.
func OpenURL(ctx context.Context, h platform.Host, u string) error {
	cu, free := cstr(u)
	defer free()
	C.giodeOpenURL(cu)
	return nil
}

// OpenWith shows the "Open in…" menu, or a Quick Look preview.
func OpenWith(ctx context.Context, h platform.Host, path, mime string) error {
	vc, err := viewController(h)
	if err != nil {
		return err
	}
	cp, free := cstr(path)
	defer free()
	C.giodeOpenIn(vc, cp)
	return nil
}

// ShareText opens the share sheet (a popover on iPad).
func ShareText(ctx context.Context, h platform.Host, text, subject, title string) error {
	vc, err := viewController(h)
	if err != nil {
		return err
	}
	ct, f1 := cstr(text)
	defer f1()
	cs, f2 := cstr(subject)
	defer f2()
	C.giodeShare(vc, ct, nil, cs)
	return nil
}

// ShareFiles opens the share sheet for files.
func ShareFiles(ctx context.Context, h platform.Host, files []File, mime, title string) error {
	if len(files) == 0 {
		return nil
	}
	vc, err := viewController(h)
	if err != nil {
		return err
	}
	if len(files) == 1 {
		cp, free := cstr(files[0].Path)
		defer free()
		C.giodeShare(vc, nil, cp, nil)
		return nil
	}
	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = f.Path
	}
	cp, free := cstr(strings.Join(paths, "\n"))
	defer free()
	C.giodeShareFiles(vc, cp)
	return nil
}

// SaveToDownloads copies the file into the app's Documents/<folder>.
func SaveToDownloads(ctx context.Context, path, mime, folder string) (string, error) {
	dir, err := dirs.Documents()
	if err != nil {
		return "", err
	}
	if folder != "" {
		dir = filepath.Join(dir, folder)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	dst := platform.UniquePath(filepath.Join(dir, filepath.Base(path)))
	return dst, platform.CopyFile(path, dst)
}

// OpenDownloads opens the app's folder in the Files app.
func OpenDownloads(ctx context.Context, h platform.Host, folder string) error {
	dir, err := dirs.Documents()
	if err != nil {
		return err
	}
	if folder != "" {
		if sub := filepath.Join(dir, folder); isDir(sub) {
			dir = sub
		}
	}
	return OpenFolder(ctx, h, dir)
}

// OpenFolder opens a folder in the Files app with its shareddocuments
// URL scheme.
func OpenFolder(ctx context.Context, h platform.Host, path string) error {
	u := url.URL{Scheme: "shareddocuments", Path: path}
	return OpenURL(ctx, h, u.String())
}
