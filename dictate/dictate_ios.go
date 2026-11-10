package dictate

/*
#cgo CFLAGS: -Werror -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework Foundation -framework UIKit -framework AVFoundation -framework Speech

#include <stdlib.h>

void giodeDictate(int id, const char *lang);
void giodeStopDictation(int id);
int giodeDictateHasInfoKey(const char *key);
*/
import "C"

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"unsafe"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/internal/dictation"
)

func init() {
	dictation.Start = start
}

// reqs holds the running dictations' emit functions by id.
var reqs struct {
	sync.Mutex
	next int
	emit map[int]func(text string, final bool, err error)
}

func start(lang string, emit func(text string, final bool, err error)) (func(), error) {
	var missing []string
	for _, k := range []string{"NSMicrophoneUsageDescription", "NSSpeechRecognitionUsageDescription"} {
		ck := C.CString(k)
		ok := C.giodeDictateHasInfoKey(ck) != 0
		C.free(unsafe.Pointer(ck))
		if !ok {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("%w: Info.plist lacks %s (ios.usage.microphone and ios.usage.speech in giode.toml)", giode.ErrNotSupported, strings.Join(missing, ", "))
	}
	reqs.Lock()
	if reqs.emit == nil {
		reqs.emit = map[int]func(string, bool, error){}
	}
	reqs.next++
	id := reqs.next
	reqs.emit[id] = emit
	reqs.Unlock()
	cl := C.CString(lang)
	C.giodeDictate(C.int(id), cl)
	C.free(unsafe.Pointer(cl))
	return func() { C.giodeStopDictation(C.int(id)) }, nil
}

//export giodeSpeech
func giodeSpeech(id C.int, text *C.char, final C.int, errMsg *C.char) {
	reqs.Lock()
	emit := reqs.emit[int(id)]
	if final != 0 {
		delete(reqs.emit, int(id))
	}
	reqs.Unlock()
	if emit == nil {
		return
	}
	var err error
	if errMsg != nil {
		err = errors.New(C.GoString(errMsg))
	}
	emit(C.GoString(text), final != 0, err)
}
