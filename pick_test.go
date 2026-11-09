package giode

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestPickExtensions(t *testing.T) {
	got := pickExtensions([]string{"application/pdf", "md", ".TXT"})
	if !slices.Contains(got, ".pdf") || !slices.Contains(got, ".md") || !slices.Contains(got, ".txt") {
		t.Errorf("extensions %v", got)
	}
	if got := pickExtensions([]string{"image/*"}); !slices.Contains(got, ".heic") || !slices.Contains(got, ".png") {
		t.Errorf("image/* gave %v", got)
	}
	if got := pickExtensions([]string{"*/*", "md"}); got != nil {
		t.Errorf("*/* gave %v, want any file", got)
	}
}

type namedReader struct {
	io.Reader
	name string
	size int64
}

func (n namedReader) Close() error { return nil }
func (n namedReader) Name() string { return n.name }
func (n namedReader) Size() int64  { return n.size }

func TestDescribe(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.md")
	os.WriteFile(path, []byte("# hi"), 0o600)
	f, _ := os.Open(path)
	defer f.Close()
	p := describe(f)
	if p.Name != "notes.md" || p.Size != 4 || !strings.HasPrefix(p.MIME, "text/markdown") || p.Path != path {
		t.Errorf("os.File: %+v", p)
	}

	p = describe(namedReader{Reader: strings.NewReader("%PDF-1.7 ..."), name: "content://x/Q3 report.pdf", size: 12})
	if p.Name != "Q3 report.pdf" || p.Size != 12 || p.MIME != "application/pdf" || p.Path != "" {
		t.Errorf("picker file: %+v", p)
	}

	// No extension: sniffed, and the sniffed bytes are still readable.
	p = describe(namedReader{Reader: strings.NewReader("\x89PNG\r\n\x1a\nrest"), name: "Document 1", size: -1})
	data, _ := io.ReadAll(p)
	if p.MIME != "image/png" || !bytes.HasPrefix(data, []byte("\x89PNG")) || p.Size != -1 {
		t.Errorf("sniffed: %+v, data %q", p, data)
	}
}

func TestToJPEG(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 400, 200))
	for i := range img.Pix {
		img.Pix[i] = 0xff
	}
	img.Set(10, 10, color.Black)
	var buf bytes.Buffer
	png.Encode(&buf, img)
	out, err := toJPEG(&buf, 100)
	if err != nil {
		t.Fatal(err)
	}
	dec, err := jpeg.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	if b := dec.Bounds(); b.Dx() != 100 || b.Dy() != 50 {
		t.Errorf("scaled to %v, want 100x50", b)
	}
	if f := jpegFile("IMG_1.HEIC", out); f.Name != "IMG_1.jpg" || f.MIME != "image/jpeg" {
		t.Errorf("jpegFile = %+v", f)
	}
}

func TestTypeByExtension_NoSystemTable(t *testing.T) {
	// These must not depend on the system's mime.types, which Windows
	// and the phones don't have.
	for ext, want := range map[string]string{".md": "text/markdown", ".WAV": "audio/wav", ".heic": "image/heic"} {
		if got := typeByExtension(ext); !strings.HasPrefix(got, want) {
			t.Errorf("typeByExtension(%q) = %q, want %q", ext, got, want)
		}
	}
	if got := pickExtensions([]string{"text/markdown"}); !slices.Contains(got, ".md") {
		t.Errorf("text/markdown gave %v", got)
	}
}

func TestSplitPaths(t *testing.T) {
	got := splitPaths([]byte("/Users/me/a b.txt\x00/Users/me/line\nbreak.pdf\x00"))
	if !slices.Equal(got, []string{"/Users/me/a b.txt", "/Users/me/line\nbreak.pdf"}) {
		t.Errorf("splitPaths: %q", got)
	}
	if got := splitPaths(nil); got != nil {
		t.Errorf("no paths: %q", got)
	}
}
