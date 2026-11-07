//go:build windows

package giode

import (
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	procPlaySoundW = windows.NewLazySystemDLL("winmm.dll").NewProc("PlaySoundW")

	// playing holds the sound PlaySound plays from memory: Windows reads
	// it while it plays, so it's kept until the next sound replaces it.
	playingMu sync.Mutex
	playing   []byte
)

const (
	sndAsync     = 0x0001
	sndNoDefault = 0x0002
	sndMemory    = 0x0004
)

func playSound(wav []byte, _ wavInfo) error {
	if err := procPlaySoundW.Find(); err != nil {
		return ErrNotSupported
	}
	buf := append([]byte(nil), wav...)
	playingMu.Lock()
	defer playingMu.Unlock()
	// A new sound stops the one playing before this returns, so the old
	// buffer can go after it.
	r, _, _ := procPlaySoundW.Call(uintptr(unsafe.Pointer(&buf[0])), 0, sndMemory|sndAsync|sndNoDefault)
	if r == 0 {
		return errNotWAV
	}
	playing = buf
	return nil
}
