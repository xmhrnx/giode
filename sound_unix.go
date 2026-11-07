//go:build !windows && !android && !ios && !js && !darwin

package giode

import (
	"os"
	"os/exec"
	"sync"
)

// players play a WAV file: PipeWire's, PulseAudio's, then ALSA's.
var players = [][]string{{"pw-play"}, {"paplay"}, {"aplay", "-q"}}

var (
	playerOnce sync.Once
	player     []string
)

func playSound(wav []byte, _ wavInfo) error {
	playerOnce.Do(func() {
		for _, p := range players {
			if path, err := exec.LookPath(p[0]); err == nil {
				player = append([]string{path}, p[1:]...)
				return
			}
		}
	})
	if player == nil {
		return ErrNotSupported
	}
	f, err := os.CreateTemp("", "giode-sound-*.wav")
	if err != nil {
		return err
	}
	if _, err := f.Write(wav); err != nil {
		f.Close()
		os.Remove(f.Name())
		return err
	}
	f.Close()
	cmd := exec.Command(player[0], append(player[1:], f.Name())...)
	if err := cmd.Start(); err != nil {
		os.Remove(f.Name())
		return err
	}
	go func() {
		cmd.Wait()
		os.Remove(f.Name())
	}()
	return nil
}
