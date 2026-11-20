package sharetarget

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
	defer f.File.Close()
	if f.Name != "notes.txt" || f.Size != -1 {
		t.Errorf("file = %+v", f)
	}
	if b, _ := io.ReadAll(f.File); string(b) != "hello" {
		t.Errorf("content = %q", b)
	}
}
