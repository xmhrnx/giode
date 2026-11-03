package giode

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestReadInboxShare(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "1.bin"), []byte("hello"), 0o644)
	os.WriteFile(filepath.Join(dir, "share.json"), []byte(`{"text":"look","subject":"Hi",
		"files":[{"file":"1.bin","name":"notes.txt"},{"file":"../escape","name":"x"},{"file":"missing"}]}`), 0o644)
	c, ok := readInboxShare(dir)
	if !ok || c.Text != "look" || c.Subject != "Hi" || len(c.Files) != 1 {
		t.Fatalf("got %+v, %v", c, ok)
	}
	f := c.Files[0]
	defer f.Close()
	if f.Name != "notes.txt" || f.MIME != "text/plain; charset=utf-8" || f.Size != 5 || f.Path != "" {
		t.Errorf("file = %+v", f)
	}
	if b, _ := io.ReadAll(f); string(b) != "hello" {
		t.Errorf("content = %q", b)
	}
}

func TestSharedFileSniffs(t *testing.T) {
	p := filepath.Join(t.TempDir(), "blob")
	os.WriteFile(p, []byte("%PDF-1.7\n..."), 0o644)
	f, _ := os.Open(p)
	sf := sharedFile(f, "blob", "", -1)
	defer sf.Close()
	if sf.MIME != "application/pdf" || sf.Size != 12 {
		t.Errorf("got %+v", sf)
	}
	if b, _ := io.ReadAll(sf); string(b) != "%PDF-1.7\n..." {
		t.Errorf("sniffing lost bytes: %q", b)
	}
}
