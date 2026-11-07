//go:build darwin && !ios

package giode

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework AppKit

int giodePlaySound(const void *data, int n);
*/
import "C"

import "unsafe"

func playSound(wav []byte, _ wavInfo) error {
	if C.giodePlaySound(unsafe.Pointer(&wav[0]), C.int(len(wav))) == 0 {
		return errNotWAV
	}
	return nil
}
