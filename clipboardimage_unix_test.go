//go:build !windows && !android && !ios && !js && !darwin

package giode

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"testing"
	"time"
)

// Copies a PNG with xclip and reads it back. Runs where an X display
// and xclip are available (CI: under xvfb-run).
func TestReadClipboardImageX11(t *testing.T) {
	xclip, err := exec.LookPath("xclip")
	if err != nil || os.Getenv("DISPLAY") == "" {
		t.Skip("needs xclip and an X display")
	}
	t.Setenv("WAYLAND_DISPLAY", "")
	src := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	src.SetNRGBA(2, 1, color.NRGBA{R: 200, G: 10, B: 30, A: 255})
	var buf bytes.Buffer
	png.Encode(&buf, src)
	// xclip keeps serving the selection from the background.
	cmd := exec.Command(xclip, "-selection", "clipboard", "-t", "image/png", "-i")
	cmd.Stdin = &buf
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	img, err := (&App{}).ReadClipboardImage(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if img == nil || img.Bounds() != src.Bounds() {
		t.Fatalf("read %v", img)
	}
	if r, g, b, _ := img.At(2, 1).RGBA(); r>>8 != 200 || g>>8 != 10 || b>>8 != 30 {
		t.Errorf("pixel (%d,%d,%d)", r>>8, g>>8, b>>8)
	}

	// Text on the clipboard: no image.
	cmd = exec.Command(xclip, "-selection", "clipboard", "-i")
	cmd.Stdin = bytes.NewBufferString("hello")
	cmd.Run()
	time.Sleep(200 * time.Millisecond)
	if img, err := (&App{}).ReadClipboardImage(context.Background()); img != nil || err != nil {
		t.Errorf("text clipboard: %v, %v; want no image", img, err)
	}
}
