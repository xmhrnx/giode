package markdown

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/fonts"
)

func layoutDoc(src string, o Options, width int) (layout.Dimensions, []string) {
	elements.BeginFrame()
	elements.BeginSemantics()
	var ops op.Ops
	dims := New(src, o).Layout(layout.Context{Ops: &ops, Now: time.Now(), Constraints: layout.Constraints{Max: image.Pt(width, 1e6)}})
	var labels []string
	for _, n := range elements.EndSemantics() {
		labels = append(labels, n.Label)
	}
	return dims, labels
}

// Images inside a paragraph load, show their alt text until then, and
// take their size, narrowed to the width.
func TestMarkdownImages(t *testing.T) {
	fonts.UseGofont()
	elements.SetWindowSize(image.Pt(600, 2000))
	defer elements.SetWindowSize(image.Point{})
	release := make(chan struct{})
	load := func(src string) (image.Image, error) {
		<-release
		switch src {
		case "wide.png":
			return image.NewNRGBA(image.Rect(0, 0, 1000, 500)), nil
		case "small.png":
			return image.NewNRGBA(image.Rect(0, 0, 40, 30)), nil
		}
		return nil, errors.New("not found")
	}
	src := "Before ![A wide picture](wide.png) after, ![missing one](nope.png) and ![](small.png).\n"
	o := Options{LoadImage: load}
	_, labels := layoutDoc(src, o, 600)
	if !contains(labels, "A wide picture") {
		t.Errorf("alt text not shown while loading: %q", labels)
	}
	close(release)
	deadline := time.Now().Add(2 * time.Second)
	for {
		imageMu.Lock()
		done := imageCache["wide.png"].done && imageCache["nope.png"].done && imageCache["small.png"].done
		imageMu.Unlock()
		if done || time.Now().After(deadline) {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	dims, labels := layoutDoc(src, o, 600)
	if contains(labels, "A wide picture") || !contains(labels, "missing one") {
		t.Errorf("after loading: %q", labels)
	}
	// The wide picture is narrowed to 600 wide, 300 high; text and the
	// small picture's row add to that.
	if dims.Size.Y < 300+30 || dims.Size.Y > 300+30+200 {
		t.Errorf("height %d", dims.Size.Y)
	}
	if !contains(labels, "Before") || containsExact(labels, "Before ") {
		t.Errorf("text before the image lost: %q", labels)
	}

	// Without a loader, images are their alt text in the paragraph.
	_, labels = layoutDoc(src, Options{}, 600)
	if len(labels) == 0 {
		t.Fatal("no text")
	}
}

func contains(labels []string, s string) bool {
	for _, l := range labels {
		if bytes.Contains([]byte(l), []byte(s)) {
			return true
		}
	}
	return false
}

func TestFetchImage(t *testing.T) {
	var buf bytes.Buffer
	png.Encode(&buf, image.NewNRGBA(image.Rect(0, 0, 3, 2)))
	img, err := FetchImage("data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()))
	if err != nil || img.Bounds().Dx() != 3 {
		t.Fatalf("data: URI: %v", err)
	}
	p := filepath.Join(t.TempDir(), "a b.png")
	os.WriteFile(p, buf.Bytes(), 0o600)
	if img, err := FetchImage(p); err != nil || img.Bounds().Dy() != 2 {
		t.Errorf("file: %v", err)
	}
	if _, err := FetchImage("data:text/plain,hi"); err == nil {
		t.Error("non-base64 data: URI read")
	}
	big := image.NewNRGBA(image.Rect(0, 0, 4096, 1024))
	if b := boundImage(big).Bounds(); b.Dx() != maxImageSide || b.Dy() != 512 {
		t.Errorf("bounded to %v", b)
	}
}

func containsExact(labels []string, s string) bool {
	for _, l := range labels {
		if l == s {
			return true
		}
	}
	return false
}
