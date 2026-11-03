package giode

import "testing"

func TestFilePattern(t *testing.T) {
	if got := filePattern([]string{".wav", ".mp3", "*.flac"}); got != "*.wav;*.mp3;*.flac" {
		t.Errorf("filePattern = %q", got)
	}
}
