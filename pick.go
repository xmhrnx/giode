package giode

import (
	"bufio"
	"bytes"
	"context"
	"image"
	"image/jpeg"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	_ "image/gif"
	_ "image/png"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"

	"github.com/xmhrnx/giode/internal/optin"
)

// PickedFile is a file the user picked, open for reading, with what's
// known about it.
type PickedFile struct {
	io.ReadCloser
	// Name is the display name ("Q3 report.pdf"), without a directory.
	Name string
	// MIME is the content type, from the name's extension or, failing
	// that, the first bytes of the content.
	MIME string
	// Size is the length in bytes, or -1 when unknown.
	Size int64
	// Path is the file's absolute path on desktops, for work that
	// needs the file itself rather than a stream (opening a disk image
	// at random offsets, reopening it later); on iOS, the path of the
	// picker's copy in the app's temporary directory. It's empty on
	// Android and the web, whose pickers give content without a path.
	Path string
}

// PickOptions configures PickFiles.
type PickOptions struct {
	// Types limits the files offered, as MIME types ("application/pdf",
	// "image/*") or extensions ("md", ".txt"). Empty allows any file.
	Types []string
	// Multiple lets the user pick several files.
	Multiple bool
}

// commonExts expands MIME wildcards, which the pickers take as
// extensions.
var commonExts = map[string][]string{
	"image": {".png", ".jpg", ".jpeg", ".gif", ".webp", ".heic", ".heif", ".bmp"},
	"audio": {".mp3", ".wav", ".m4a", ".aac", ".ogg", ".flac", ".opus"},
	"video": {".mp4", ".mov", ".webm", ".mkv", ".m4v"},
	"text":  {".txt", ".md", ".csv", ".json", ".html", ".xml"},
}

// pickExtensions converts PickOptions.Types to extensions.
func pickExtensions(types []string) []string {
	var exts []string
	for _, t := range types {
		t = strings.ToLower(strings.TrimSpace(t))
		switch {
		case t == "" || t == "*/*":
			return nil
		case strings.HasSuffix(t, "/*"):
			exts = append(exts, commonExts[strings.TrimSuffix(t, "/*")]...)
		case strings.Contains(t, "/"):
			for ext, kt := range knownTypes {
				if strings.HasPrefix(kt, t) {
					exts = append(exts, ext)
				}
			}
			if e, _ := mime.ExtensionsByType(t); len(e) > 0 {
				exts = append(exts, e...)
			}
		default:
			if !strings.HasPrefix(t, ".") {
				t = "." + t
			}
			exts = append(exts, t)
		}
	}
	return exts
}

// PickFiles shows the platform's file picker (see OpenFile) and returns
// the picked files with their names, types and sizes - the display name
// and size from the picker on Android and iOS, from the file itself on
// desktops. Like OpenFile it blocks until the user decides: call it from
// a goroutine. It returns ErrCanceled when the user closes the picker.
//
//	files, err := app.PickFiles(ctx, giode.PickOptions{Types: []string{"image/*", "application/pdf"}, Multiple: true})
func (a *App) PickFiles(ctx context.Context, opts PickOptions) ([]PickedFile, error) {
	exts := pickExtensions(opts.Types)
	var raw []io.ReadCloser
	if opts.Multiple {
		fs, err := a.OpenFiles(ctx, FileFilter{Extensions: exts})
		if err != nil {
			return nil, err
		}
		raw = fs
	} else {
		f, err := a.OpenFile(ctx, FileFilter{Extensions: exts})
		if err != nil {
			return nil, err
		}
		raw = []io.ReadCloser{f}
	}
	files := make([]PickedFile, len(raw))
	for i, f := range raw {
		files[i] = describe(f)
	}
	return files, nil
}

// knownTypes covers common files Go's built-in MIME table lacks; the
// system table that fills the gap on Linux isn't there on Windows or
// the phones.
var knownTypes = map[string]string{
	".md":       "text/markdown; charset=utf-8",
	".markdown": "text/markdown; charset=utf-8",
	".txt":      "text/plain; charset=utf-8",
	".csv":      "text/csv; charset=utf-8",
	".heic":     "image/heic",
	".heif":     "image/heif",
	".mp3":      "audio/mpeg",
	".wav":      "audio/wav",
	".m4a":      "audio/mp4",
	".ogg":      "audio/ogg",
	".flac":     "audio/flac",
	".mp4":      "video/mp4",
	".mov":      "video/quicktime",
	".zip":      "application/zip",
	".docx":     "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xlsx":     "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".pptx":     "application/vnd.openxmlformats-officedocument.presentationml.presentation",
}

// typeByExtension returns the MIME type for a file extension, or "".
func typeByExtension(ext string) string {
	ext = strings.ToLower(ext)
	if t, ok := knownTypes[ext]; ok {
		return t
	}
	return mime.TypeByExtension(ext)
}

// describe fills in what's known about a picked file.
func describe(f io.ReadCloser) PickedFile {
	p := PickedFile{ReadCloser: f, Size: -1}
	switch v := f.(type) {
	case *os.File:
		p.Name = filepath.Base(v.Name())
		if abs, err := filepath.Abs(v.Name()); err == nil {
			p.Path = abs
		}
		if st, err := v.Stat(); err == nil {
			p.Size = st.Size()
		}
	case interface {
		Name() string
		Size() int64
	}:
		// The pickers' files on Android and iOS.
		p.Name = filepath.Base(v.Name())
		if s := v.Size(); s >= 0 {
			p.Size = s
		}
	}
	if t := typeByExtension(filepath.Ext(p.Name)); t != "" {
		p.MIME = t
		return p
	}
	// Sniff the content, keeping what was read.
	br := bufio.NewReaderSize(f, 512)
	head, _ := br.Peek(512)
	p.MIME = http.DetectContentType(head)
	p.ReadCloser = struct {
		io.Reader
		io.Closer
	}{br, f}
	return p
}

// PhotoSource selects where PickPhoto takes the photo from.
type PhotoSource int

const (
	// PhotoLibrary picks an existing photo: the system photo picker on
	// Android, an image file elsewhere.
	PhotoLibrary PhotoSource = iota
	// Camera takes a new photo with the camera: Android, and iOS when
	// the app's Info.plist has NSCameraUsageDescription. It needs the
	// github.com/xmhrnx/giode/takephoto package imported, so that only
	// apps that take photos carry the camera code.
	Camera
)

// PhotoOptions configures PickPhoto.
type PhotoOptions struct {
	Source PhotoSource
	// MaxSide scales the photo down so its longer side is at most this
	// many pixels; 0 keeps its size.
	MaxSide int
}

// PickPhoto lets the user pick a photo (or take one, with Camera) and
// returns it as a JPEG, scaled to MaxSide and rotated upright. On
// Android it uses the system photo picker (Android 13+, or the image
// picker before that) and decodes HEIC there, so HEIC never reaches the
// app. On desktops it picks an image file (JPEG, PNG, GIF, WebP) and
// re-encodes it. It blocks until the user decides: call it from a
// goroutine. It returns ErrCanceled when the user backs out.
func (a *App) PickPhoto(ctx context.Context, opts PhotoOptions) (PickedFile, error) {
	if opts.Source == Camera && !optin.TakePhotoLinked {
		return PickedFile{}, notLinked("takephoto")
	}
	return a.pickPhoto(ctx, opts)
}

// photoFromFile is PickPhoto through the file picker, decoding and
// re-encoding in Go.
func (a *App) photoFromFile(ctx context.Context, opts PhotoOptions) (PickedFile, error) {
	if opts.Source == Camera {
		return PickedFile{}, ErrNotSupported
	}
	files, err := a.PickFiles(ctx, PickOptions{Types: []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}})
	if err != nil {
		return PickedFile{}, err
	}
	f := files[0]
	defer f.Close()
	data, err := toJPEG(f, opts.MaxSide)
	if err != nil {
		return PickedFile{}, err
	}
	return jpegFile(f.Name, data), nil
}

// toJPEG decodes an image and re-encodes it as JPEG, upright per its
// EXIF orientation and scaled so its longer side is at most maxSide.
func toJPEG(r io.Reader, maxSide int) ([]byte, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	// Turn it upright as the camera meant, as the phones' pickers do.
	img = applyOrientation(img, jpegOrientation(raw))
	b := img.Bounds()
	if long := max(b.Dx(), b.Dy()); maxSide > 0 && long > maxSide {
		w, h := b.Dx()*maxSide/long, b.Dy()*maxSide/long
		dst := image.NewRGBA(image.Rect(0, 0, max(1, w), max(1, h)))
		draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Src, nil)
		img = dst
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// jpegFile wraps JPEG bytes as a PickedFile named after the source.
func jpegFile(name string, data []byte) PickedFile {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	if base == "" {
		base = "photo"
	}
	return PickedFile{
		ReadCloser: io.NopCloser(bytes.NewReader(data)),
		Name:       base + ".jpg",
		MIME:       "image/jpeg",
		Size:       int64(len(data)),
	}
}
