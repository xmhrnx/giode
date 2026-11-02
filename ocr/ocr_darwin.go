package ocr

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework Foundation -framework Vision -framework ImageIO -framework CoreGraphics

#include <stdint.h>
#include <stdlib.h>

void giodeOCR(int id, const void *data, int n, const char *langs, int fast);
*/
import "C"

import (
	"context"
	"errors"
	"strings"
	"unsafe"

	"github.com/xmhrnx/giode/internal/bridge"
)

const supported = true

type answer struct {
	json []byte
	err  error
}

var requests bridge.Requests[answer]

func recognize(ctx context.Context, data []byte, opts Options) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("ocr: no image")
	}
	id, ch := requests.New()
	langs := C.CString(strings.Join(opts.Languages, ","))
	defer C.free(unsafe.Pointer(langs))
	fast := 0
	if opts.Fast {
		fast = 1
	}
	// Vision copies the data before giodeOCR returns.
	C.giodeOCR(C.int(id), unsafe.Pointer(&data[0]), C.int(len(data)), langs, C.int(fast))
	a, err := requests.Wait(ctx, id, ch)
	if err != nil {
		return nil, err
	}
	return a.json, a.err
}

//export giodeOCRDone
func giodeOCRDone(id C.int, js *C.char, msg *C.char) {
	var a answer
	if msg != nil {
		a.err = errors.New("ocr: " + C.GoString(msg))
	} else {
		a.json = []byte(C.GoString(js))
	}
	requests.Done(int(id), a)
}
