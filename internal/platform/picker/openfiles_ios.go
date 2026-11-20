package picker

/*
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>

int giodeOpenDocuments(CFTypeRef vc, char *exts, int multiple);
*/
import "C"

import (
	"fmt"
	"strings"
	"unsafe"

	"github.com/xmhrnx/giode/internal/platform"
)

// documentsPicked receives the paths giodeOpenDocuments' picker
// returns, nil when the user cancels it.
var documentsPicked = make(chan []string, 1)

//export giodeDocumentsPicked
func giodeDocumentsPicked(paths *C.char, n C.int) {
	if paths == nil {
		documentsPicked <- nil
		return
	}
	documentsPicked <- splitPaths(C.GoBytes(unsafe.Pointer(paths), n))
}

// openFileDialog shows UIDocumentPickerViewController for files
// (several when multiple is set; Gio's explorer can't on iOS) and
// returns the paths of the copies it makes in the app's temporary
// directory.
func openFileDialog(h platform.Host, exts []string, multiple bool) ([]string, error) {
	dialogMu.Lock()
	defer dialogMu.Unlock()
	vc, err := viewController(h)
	if err != nil {
		return nil, err
	}
	clean := make([]string, 0, len(exts))
	for _, e := range exts {
		if e = strings.TrimLeft(strings.TrimSpace(e), "*."); e != "" {
			clean = append(clean, e)
		}
	}
	cexts := C.CString(strings.Join(clean, "\n"))
	defer C.free(unsafe.Pointer(cexts))
	m := C.int(0)
	if multiple {
		m = 1
	}
	if C.giodeOpenDocuments(vc, cexts, m) == 0 {
		return nil, fmt.Errorf("%w: the document picker needs iOS 14", platform.ErrNotSupported)
	}
	paths := <-documentsPicked
	if len(paths) == 0 {
		return nil, platform.ErrCanceled
	}
	return paths, nil
}
