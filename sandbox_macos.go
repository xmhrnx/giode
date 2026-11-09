//go:build darwin && !ios

package giode

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework AppKit -framework ServiceManagement

#include <stdlib.h>

int giodeLoginItemSet(int on, char **err);
int giodeLoginItemEnabled(void);
int giodeWorkspaceOpen(const char *target, int isPath);
int giodeWorkspaceReveal(const char *path);
void *giodeBookmark(const char *path, int *n);
char *giodeResolveBookmark(const void *data, int n, int *stale);
*/
import "C"

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"unsafe"
)

// The Mac App Store's apps run in the App Sandbox: the parts of giode
// that touch the system outside the app go through the APIs the sandbox
// allows.

// sandboxed reports whether the app runs in the App Sandbox.
func sandboxed() bool {
	return os.Getenv("APP_SANDBOX_CONTAINER_ID") != ""
}

// setLoginItem registers the app as a login item with SMAppService,
// which a sandboxed app may use where it may not write a LaunchAgent:
// macOS 13 and later; ErrNotSupported before.
func setLoginItem(on bool) error {
	var cerr *C.char
	o := C.int(0)
	if on {
		o = 1
	}
	switch C.giodeLoginItemSet(o, &cerr) {
	case 1:
		return nil
	case -1:
		return ErrNotSupported
	}
	defer C.free(unsafe.Pointer(cerr))
	return errors.New("giode: start at login: " + C.GoString(cerr))
}

func loginItemEnabled() bool {
	return C.giodeLoginItemEnabled() == 1
}

// workspaceOpen opens a URL, or a path when isPath is set, with
// NSWorkspace, which the sandbox allows where running open(1) isn't.
func workspaceOpen(target string, isPath bool) error {
	if isPath {
		// A container's Downloads is a link to the user's; Finder shows
		// the real folder.
		if real, err := filepath.EvalSymlinks(target); err == nil {
			target = real
		}
	}
	c := C.CString(target)
	defer C.free(unsafe.Pointer(c))
	p := C.int(0)
	if isPath {
		p = 1
	}
	if C.giodeWorkspaceOpen(c, p) == 0 {
		return errors.New("giode: couldn't open " + target)
	}
	return nil
}

// revealFile shows path selected in a Finder window.
func revealFile(path string) error {
	if real, err := filepath.EvalSymlinks(path); err == nil {
		path = real
	}
	c := C.CString(path)
	defer C.free(unsafe.Pointer(c))
	if C.giodeWorkspaceReveal(c) == 0 {
		return errors.New("giode: couldn't show " + path)
	}
	return nil
}

// Folders the user chose (PickFolder) stay reachable only until a
// sandboxed app quits; a security-scoped bookmark brings them back. giode
// keeps one for each, in the app's container, and opens them all at
// start, so a download folder chosen once keeps working.

var accessMu sync.Mutex

func folderAccessFile() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "giode", "folder-access.json"), nil
}

func readFolderAccess() map[string][]byte {
	marks := map[string][]byte{}
	if p, err := folderAccessFile(); err == nil {
		if b, err := os.ReadFile(p); err == nil {
			json.Unmarshal(b, &marks)
		}
	}
	return marks
}

func writeFolderAccess(marks map[string][]byte) {
	p, err := folderAccessFile()
	if err != nil {
		return
	}
	b, err := json.Marshal(marks)
	if err != nil {
		return
	}
	os.MkdirAll(filepath.Dir(p), 0o700)
	os.WriteFile(p, b, 0o600)
}

func bookmark(path string) []byte {
	c := C.CString(path)
	defer C.free(unsafe.Pointer(c))
	var n C.int
	data := C.giodeBookmark(c, &n)
	if data == nil {
		return nil
	}
	defer C.free(data)
	return C.GoBytes(data, n)
}

// rememberFolderAccess keeps a bookmark of a folder the user chose, in a
// sandboxed app. It needs the com.apple.security.files.bookmarks.app-scope
// entitlement (giode build mas adds it); without it, it does nothing.
func rememberFolderAccess(path string) {
	if !sandboxed() || path == "" {
		return
	}
	b := bookmark(path)
	if b == nil {
		return
	}
	accessMu.Lock()
	defer accessMu.Unlock()
	marks := readFolderAccess()
	marks[path] = b
	writeFolderAccess(marks)
}

// restoreFolderAccess opens the bookmarked folders for this run,
// renewing stale bookmarks and forgetting those that no longer resolve.
func restoreFolderAccess() {
	if !sandboxed() {
		return
	}
	accessMu.Lock()
	defer accessMu.Unlock()
	marks := readFolderAccess()
	changed := false
	for path, b := range marks {
		if len(b) == 0 {
			delete(marks, path)
			changed = true
			continue
		}
		var stale C.int
		resolved := C.giodeResolveBookmark(unsafe.Pointer(&b[0]), C.int(len(b)), &stale)
		if resolved == nil {
			delete(marks, path)
			changed = true
			continue
		}
		got := C.GoString(resolved)
		C.free(unsafe.Pointer(resolved))
		if stale != 0 || got != path {
			delete(marks, path)
			if nb := bookmark(got); nb != nil {
				marks[got] = nb
			}
			changed = true
		}
	}
	if changed {
		writeFolderAccess(marks)
	}
}

func init() {
	restoreFolderAccess()
}
