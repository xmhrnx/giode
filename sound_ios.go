//go:build ios

package giode

/*
int giodePlaySound(const void *data, int n);
*/
import "C"

import "unsafe"

// playSound plays through AVAudioPlayer (platform_ios.m), in the
// ambient session: mixed with other audio, and silent in silent mode.
func playSound(wav []byte, _ wavInfo) error {
	if C.giodePlaySound(unsafe.Pointer(&wav[0]), C.int(len(wav))) == 0 {
		return errNotWAV
	}
	return nil
}
