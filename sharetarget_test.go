package giode

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

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
