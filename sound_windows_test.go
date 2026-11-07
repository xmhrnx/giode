//go:build windows

package giode

import "testing"

// Windows plays the WAV from memory without waiting; a buffer that
// isn't a WAV is refused before it gets there.
func TestPlaySoundWindows(t *testing.T) {
	samples := make([]float32, 4410)
	if err := (&App{}).PlaySound(EncodeWAV(samples, 44100)); err != nil && err != ErrNotSupported {
		t.Fatalf("PlaySound: %v", err)
	}
}
