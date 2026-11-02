package giode

import (
	"context"
	"path/filepath"
	"strings"
)

// Shareable is something App.Share can hand to the system: ShareText,
// ShareFile or ShareFiles.
type Shareable interface {
	share(ctx context.Context, a *App) error
}

// ShareText shares text, such as a link or a note.
type ShareText struct {
	Text string
	// Subject is used by apps that take one, such as email. Optional.
	Subject string
	// Title heads the share sheet. Optional.
	Title string
}

// ShareFile shares the file at Path.
type ShareFile struct {
	Path string
	// MIME is the file's type, such as "audio/wav". Guessed from the
	// extension when empty.
	MIME string
	// Title heads the share sheet. Optional.
	Title string
}

// ShareFiles shares several files at once: the pages of a scan as
// JPEGs, a batch of exports. On desktops it shows the first in the file
// manager.
type ShareFiles struct {
	Paths []string
	// MIME is the files' common type, such as "image/jpeg". When empty
	// it's guessed from the extensions: their shared type, else the
	// shared kind ("image/*"), else any.
	MIME string
	// Title heads the share sheet. Optional.
	Title string
}

func (f ShareFiles) mimeType() string {
	if f.MIME != "" {
		return f.MIME
	}
	common := ""
	for i, p := range f.Paths {
		t := ShareFile{Path: p}.mimeType()
		switch {
		case i == 0:
			common = t
		case common == t:
		case kind(common) == kind(t):
			common = kind(t) + "/*"
		default:
			return "*/*"
		}
	}
	if common == "" || common == "application/octet-stream" {
		return "*/*"
	}
	return common
}

func kind(mime string) string {
	k, _, _ := strings.Cut(mime, "/")
	return k
}

func (f ShareFile) mimeType() string {
	if f.MIME != "" {
		return f.MIME
	}
	if t := typeByExtension(filepath.Ext(f.Path)); t != "" {
		return t
	}
	return "application/octet-stream"
}

// Share hands item to the system, the usual way to get something out
// of an app on a phone:
//
//	err := app.Share(ctx, giode.ShareFile{Path: p, MIME: "audio/wav"})
//
// On Android it opens the system share sheet. A shared file is first
// copied into the device's Downloads folder (apps built with gogio
// can't declare the FileProvider Android otherwise needs to hand
// files to other apps), which needs Android 10 or later.
//
// On desktops, ShareText copies the text to the clipboard and
// ShareFile shows the file in the file manager. On iOS it opens the
// share sheet (a popover on iPad). On the web Share returns
// ErrNotSupported.
//
// Share can take a while for a large file, so call it from a
// goroutine rather than a click handler.
func (a *App) Share(ctx context.Context, item Shareable) error {
	return item.share(ctx, a)
}
