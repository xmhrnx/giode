// Command skilldocs copies the documentation (docs/, without the
// contributor-only pages) into the agent skill's references folder,
// skills/giode/references, so the skill carries the docs wherever it's
// installed. Run it through go generate in the repository root.
package main

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// skipped are docs pages that don't help building apps.
var skipped = map[string]bool{"README.md": true, "contributing.md": true, "architecture.md": true}

func main() {
	if err := sync("docs", filepath.Join("skills", "giode", "references")); err != nil {
		log.Fatal(err)
	}
}

// pages returns the docs pages to copy, as slash paths relative to docs.
func pages(docs string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(docs, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
			return err
		}
		rel, err := filepath.Rel(docs, path)
		if err != nil {
			return err
		}
		if !skipped[filepath.Base(rel)] {
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	return out, err
}

// sync makes refs an exact copy of the pages in docs.
func sync(docs, refs string) error {
	if err := os.RemoveAll(refs); err != nil {
		return err
	}
	ps, err := pages(docs)
	if err != nil {
		return err
	}
	for _, p := range ps {
		b, err := os.ReadFile(filepath.Join(docs, p))
		if err != nil {
			return err
		}
		dst := filepath.Join(refs, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, b, 0o644); err != nil {
			return err
		}
	}
	return nil
}
