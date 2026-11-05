//go:build !windows && !android && !ios && !js && !darwin

package giode

import (
	"bytes"
	"context"
	"errors"
	"image"
	"io"
	"os"
	"os/exec"
	"strings"
)

// imageTypes are the clipboard types tried, most faithful first.
var imageTypes = []string{"image/png", "image/webp", "image/jpeg", "image/gif", "image/bmp", "image/tiff"}

// readClipboardImage reads the clipboard through wl-paste (Wayland) or
// xclip (X11): Gio's own clipboard only carries text.
func (a *App) readClipboardImage(ctx context.Context) (image.Image, error) {
	var list, read func(t string) *exec.Cmd
	wl, wlErr := exec.LookPath("wl-paste")
	xc, xcErr := exec.LookPath("xclip")
	switch {
	case os.Getenv("WAYLAND_DISPLAY") != "" && wlErr == nil:
		list = func(string) *exec.Cmd { return exec.CommandContext(ctx, wl, "--list-types") }
		read = func(t string) *exec.Cmd { return exec.CommandContext(ctx, wl, "--no-newline", "--type", t) }
	case xcErr == nil:
		list = func(string) *exec.Cmd {
			return exec.CommandContext(ctx, xc, "-selection", "clipboard", "-t", "TARGETS", "-o")
		}
		read = func(t string) *exec.Cmd {
			return exec.CommandContext(ctx, xc, "-selection", "clipboard", "-t", t, "-o")
		}
	case wlErr == nil:
		list = func(string) *exec.Cmd { return exec.CommandContext(ctx, wl, "--list-types") }
		read = func(t string) *exec.Cmd { return exec.CommandContext(ctx, wl, "--no-newline", "--type", t) }
	default:
		return nil, errors.Join(ErrNotSupported, errors.New("giode: reading an image from the clipboard needs wl-paste (wl-clipboard) or xclip"))
	}
	out, err := list("").Output()
	if err != nil {
		// No owner, or nothing copied: no image.
		return nil, nil
	}
	types := strings.Fields(string(out))
	for _, want := range imageTypes {
		for _, t := range types {
			if t != want {
				continue
			}
			cmd := read(t)
			var buf bytes.Buffer
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				return nil, err
			}
			if err := cmd.Start(); err != nil {
				return nil, err
			}
			_, err = io.Copy(&buf, io.LimitReader(stdout, maxClipboardImage))
			if werr := cmd.Wait(); err == nil {
				err = werr
			}
			if err != nil {
				return nil, err
			}
			return decodeClipboardImage(buf.Bytes())
		}
	}
	return nil, nil
}
