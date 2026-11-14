package settings

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app", "settings.json")
	s := Open(path)
	if s.Has("theme") {
		t.Fatal("a new store has a value")
	}
	var theme string
	if s.Get("theme", &theme) {
		t.Fatal("Get found a value in a new store")
	}
	type progress struct {
		Lesson int
		Done   map[string]bool
	}
	if err := s.Set("theme", "dark"); err != nil {
		t.Fatal(err)
	}
	if err := s.Set("progress", progress{Lesson: 3, Done: map[string]bool{"orbits": true}}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("not written: %v", err)
	}
	// Windows has no Unix permissions to check.
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Errorf("file mode %v, want 0600", info.Mode().Perm())
	}

	// Another store over the same file - the app started again.
	again := Open(path)
	var p progress
	if !again.Get("theme", &theme) || theme != "dark" {
		t.Errorf("theme %q after reopening", theme)
	}
	if !again.Get("progress", &p) || p.Lesson != 3 || !p.Done["orbits"] {
		t.Errorf("progress %+v after reopening", p)
	}
	if keys := again.Keys(); len(keys) != 2 || keys[0] != "progress" || keys[1] != "theme" {
		t.Errorf("keys %v", keys)
	}
	// A value that doesn't decode as asked isn't found.
	var n int
	if again.Get("theme", &n) {
		t.Error("a string decoded into an int")
	}
	if err := again.Delete("theme"); err != nil {
		t.Fatal(err)
	}
	if Open(path).Has("theme") {
		t.Error("deleted key still there after reopening")
	}
	// No temporary files are left beside it.
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Errorf("directory holds %d files, want only settings.json", len(entries))
	}
}

func TestDamagedFileIsDropped(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	os.WriteFile(path, []byte("{not json"), 0o600)
	s := Open(path)
	if err := s.Set("a", 1); err != nil {
		t.Fatal(err)
	}
	var a int
	if !Open(path).Get("a", &a) || a != 1 {
		t.Errorf("a = %d after writing over a damaged file", a)
	}
}

func TestBind(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	s := Open(path)
	done := BindTo(s, "done", map[string]bool{})
	if len(done.Get()) != 0 {
		t.Fatalf("initial %v", done.Get())
	}
	done.Update(func(m map[string]bool) map[string]bool {
		m = map[string]bool{"lesson1": true}
		return m
	})
	// The app starts again: the State comes back with what was saved.
	again := BindTo(Open(path), "done", map[string]bool{})
	if !again.Get()["lesson1"] {
		t.Errorf("after restarting: %v, want lesson1 done", again.Get())
	}
	// Package-level helpers on a store of their own.
	old := Default
	defer func() { Default = old }()
	Default = Open(filepath.Join(t.TempDir(), "d.json"))
	if Value("count", 7) != 7 {
		t.Error("default not returned")
	}
	Save("count", 9)
	if Value("count", 7) != 9 {
		t.Error("saved value not returned")
	}
}
