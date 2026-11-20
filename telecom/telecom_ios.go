//go:build ios

package telecom

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework Foundation -framework CallKit -framework PushKit -framework AVFoundation

#include <stdlib.h>

void giodeTelecomSetup(int video);
char *giodeTelecomIncoming(const char *uuid, const char *handle, const char *name, int video);
char *giodeTelecomStart(const char *uuid, const char *handle, const char *name, int video);
void giodeTelecomConnected(const char *uuid);
char *giodeTelecomEnd(const char *uuid);
void giodeTelecomEnded(const char *uuid, int reason);
char *giodeTelecomMute(const char *uuid, int muted);
*/
import "C"

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"unsafe"

	"github.com/xmhrnx/giode"
)

const supported = true

func cerr(msg *C.char) error {
	if msg == nil {
		return nil
	}
	defer C.free(unsafe.Pointer(msg))
	return errors.New("telecom: " + C.GoString(msg))
}

func cstrs(ss ...string) ([]*C.char, func()) {
	out := make([]*C.char, len(ss))
	for i, s := range ss {
		out[i] = C.CString(s)
	}
	return out, func() {
		for _, p := range out {
			C.free(unsafe.Pointer(p))
		}
	}
}

func b2i(b bool) C.int {
	if b {
		return 1
	}
	return 0
}

func setup(*giode.App) error {
	C.giodeTelecomSetup(1)
	return nil
}

func reportIncoming(ctx context.Context, c Call) error {
	s, free := cstrs(c.ID, c.Handle, c.Name)
	defer free()
	return cerr(C.giodeTelecomIncoming(s[0], s[1], s[2], b2i(c.Video)))
}

func startOutgoing(c Call) error {
	s, free := cstrs(c.ID, c.Handle, c.Name)
	defer free()
	return cerr(C.giodeTelecomStart(s[0], s[1], s[2], b2i(c.Video)))
}

func reportConnected(id string) error {
	s, free := cstrs(id)
	defer free()
	C.giodeTelecomConnected(s[0])
	return nil
}

func end(id string) error {
	s, free := cstrs(id)
	defer free()
	return cerr(C.giodeTelecomEnd(s[0]))
}

func reportEnded(id string, r Reason) error {
	s, free := cstrs(id)
	defer free()
	C.giodeTelecomEnded(s[0], C.int(r))
	return nil
}

func setMuted(id string, muted bool) error {
	s, free := cstrs(id)
	defer free()
	return cerr(C.giodeTelecomMute(s[0], b2i(muted)))
}

//export giodeTelecomEvent
func giodeTelecomEvent(kind, a, b *C.char) {
	dispatch(C.GoString(kind), C.GoString(a), C.GoString(b))
}

//export giodeTelecomPush
func giodeTelecomPush(payload *C.char) *C.char {
	var p map[string]any
	json.Unmarshal([]byte(C.GoString(payload)), &p)
	c := callFromPayload(p)
	clean := func(s string) string { return strings.NewReplacer("\t", " ", "\n", " ").Replace(s) }
	video := "0"
	if c.Video {
		video = "1"
	}
	return C.CString(strings.Join([]string{c.ID, clean(c.Handle), clean(c.Name), video}, "\t"))
}
