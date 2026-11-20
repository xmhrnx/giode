//go:build windows

package sound

import (
	"testing"

	"github.com/xmhrnx/giode/internal/platform"
)

// Windows plays the WAV from memory without waiting; a buffer that
// isn't a WAV is refused before it gets there.
func TestPlaySoundWindows(t *testing.T) {
	samples := make([]float32, 4410)
	if err := Play(Encode(samples, 44100)); err != nil && err != platform.ErrNotSupported {
		t.Fatalf("PlaySound: %v", err)
	}
}
