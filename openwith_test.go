package giode

import (
	"context"
	"path/filepath"
	"testing"
)

// OpenWith opens files: a folder or a missing file is an error before
// any app is asked.
func TestOpenWithChecksPath(t *testing.T) {
	a := &App{}
	dir := t.TempDir()
	if err := a.OpenWith(context.Background(), dir); err == nil {
		t.Error("opened a folder")
	}
	if err := a.OpenWith(context.Background(), filepath.Join(dir, "missing.pdf")); err == nil {
		t.Error("opened a missing file")
	}
}
