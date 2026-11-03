//go:build linux && !android

package giode

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveToDownloads(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	os.MkdirAll(filepath.Join(home, ".config"), 0o755)
	os.WriteFile(filepath.Join(home, ".config", "user-dirs.dirs"), []byte(`XDG_DOWNLOAD_DIR="$HOME/Téléchargements"`+"\n"), 0o644)
	src := filepath.Join(t.TempDir(), "photo.jpg")
	os.WriteFile(src, []byte("jpeg"), 0o644)

	a := New(Config{Title: "Easy/Send"})
	got, err := a.SaveToDownloads(context.Background(), src, "")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, "Téléchargements", "Easy-Send", "photo.jpg"); got != want {
		t.Errorf("saved to %s, want %s", got, want)
	}
	again, _ := a.SaveToDownloads(context.Background(), src, "Inbox")
	again2, _ := a.SaveToDownloads(context.Background(), src, "Inbox")
	if filepath.Base(again) != "photo.jpg" || filepath.Base(again2) != "photo (2).jpg" {
		t.Errorf("second copy named %s, then %s", again, again2)
	}
	if b, _ := os.ReadFile(again2); string(b) != "jpeg" {
		t.Errorf("copy holds %q", b)
	}
	if err := a.OpenFolder(context.Background(), src); err == nil {
		t.Error("OpenFolder accepted a file")
	}
}
