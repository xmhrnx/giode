//go:build !windows && !android && !ios && !js && !darwin

package giode

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// PlaySound hands the WAV to the first player it finds, without
// waiting for it, and removes its file after.
func TestPlaySoundUnix(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "played.wav")
	// A stand-in pw-play that copies the file it's given.
	sleep, err1 := exec.LookPath("sleep")
	cp, err2 := exec.LookPath("cp")
	if err1 != nil || err2 != nil {
		t.Skip("no sleep or cp")
	}
	script := "#!/bin/sh\n" + sleep + " 0.2\n" + cp + " \"$1\" " + out + "\n"
	if err := os.WriteFile(filepath.Join(dir, "pw-play"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	playerOnce, player = sync.Once{}, nil
	defer func() { playerOnce, player = sync.Once{}, nil }()

	wav := EncodeWAV([]float32{0, 0.25, -0.25}, 22050)
	start := time.Now()
	if err := (&App{}).PlaySound(wav); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d > 150*time.Millisecond {
		t.Errorf("PlaySound waited %v for the player", d)
	}
	var got []byte
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if b, err := os.ReadFile(out); err == nil && len(b) == len(wav) {
			got = b
			break
		}
	}
	if !bytes.Equal(got, wav) {
		t.Fatalf("the player got %d bytes, want the %d-byte WAV", len(got), len(wav))
	}
	time.Sleep(100 * time.Millisecond)
	if left, _ := filepath.Glob(filepath.Join(os.TempDir(), "giode-sound-*.wav")); len(left) > 0 {
		t.Errorf("left behind %v", left)
	}

	// No player at all.
	t.Setenv("PATH", t.TempDir())
	playerOnce, player = sync.Once{}, nil
	if err := (&App{}).PlaySound(wav); err != ErrNotSupported {
		t.Errorf("with no player: %v, want ErrNotSupported", err)
	}
}
