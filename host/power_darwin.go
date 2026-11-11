//go:build darwin && !ios

package host

/*
#cgo LDFLAGS: -framework IOKit -framework CoreFoundation
#include <stdlib.h>
#include <IOKit/pwr_mgt/IOPMLib.h>

static IOPMAssertionID giodeAssert(int display, const char *reason) {
	CFStringRef r = CFStringCreateWithCString(NULL, reason, kCFStringEncodingUTF8);
	IOPMAssertionID id = 0;
	CFStringRef type = display ? kIOPMAssertionTypeNoDisplaySleep : kIOPMAssertionTypePreventUserIdleSystemSleep;
	if (r == NULL) {
		// Not UTF-8: no reason rather than CFRelease(NULL), which crashes.
		r = CFSTR("");
		CFRetain(r);
	}
	if (IOPMAssertionCreateWithName(type, kIOPMAssertionLevelOn, r, &id) != kIOReturnSuccess) {
		id = 0;
	}
	CFRelease(r);
	return id;
}
*/
import "C"

import (
	"errors"
	"sync"
	"unsafe"
)

var screenAssertion struct {
	sync.Mutex
	id C.IOPMAssertionID
}

func assert(display bool, reason string) (C.IOPMAssertionID, error) {
	cs := C.CString(reason)
	defer C.free(unsafe.Pointer(cs))
	d := C.int(0)
	if display {
		d = 1
	}
	id := C.giodeAssert(d, cs)
	if id == 0 {
		return 0, errors.New("giode: IOPMAssertionCreateWithName failed")
	}
	return id, nil
}

// KeepScreenOn keeps the display from sleeping while on is set (an
// IOKit power assertion).
func KeepScreenOn(on bool) error {
	screenAssertion.Lock()
	defer screenAssertion.Unlock()
	switch {
	case on && screenAssertion.id == 0:
		id, err := assert(true, "Keeping the screen on")
		if err != nil {
			return err
		}
		screenAssertion.id = id
	case !on && screenAssertion.id != 0:
		C.IOPMAssertionRelease(screenAssertion.id)
		screenAssertion.id = 0
	}
	return nil
}

// KeepAwake keeps the computer from sleeping while idle until release
// is called, listed with reason in Activity Monitor and pmset.
func KeepAwake(reason string) (release func(), err error) {
	id, err := assert(false, reason)
	if err != nil {
		return func() {}, err
	}
	var once sync.Once
	return func() { once.Do(func() { C.IOPMAssertionRelease(id) }) }, nil
}
