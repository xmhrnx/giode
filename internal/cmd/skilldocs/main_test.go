package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestUpToDate fails when the skill's references don't match docs/:
// run go generate in the repository root.
func TestUpToDate(t *testing.T) {
	docs, refs := "../../../docs", "../../../skills/giode/references"
	ps, err := pages(docs)
	if err != nil {
		t.Fatal(err)
	}
	norm := func(b []byte) []byte { return bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n")) }
	want := map[string]bool{}
	for _, p := range ps {
		want[p] = true
		a, err := os.ReadFile(filepath.Join(docs, p))
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(refs, p))
		if err != nil || !bytes.Equal(norm(a), norm(b)) {
			t.Errorf("skills/giode/references/%s is stale or missing: run go generate in the repository root", p)
		}
	}
	got, _ := pages(refs)
	for _, p := range got {
		if !want[p] {
			t.Errorf("skills/giode/references/%s has no page in docs/: run go generate in the repository root", p)
		}
	}
}
