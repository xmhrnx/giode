package picker

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"

	"gioui.org/app"

	"github.com/xmhrnx/giode/internal/mobile"
	"github.com/xmhrnx/giode/internal/platform"
)

// androidFile is a picked document: a stream on its descriptor, with
// the display name and size the picker gave.
type androidFile struct {
	*os.File
	name string
	size int64
}

func (f androidFile) Name() string { return f.name }
func (f androidFile) Size() int64  { return f.size }

// chooseFiles shows the system's document picker (ACTION_OPEN_DOCUMENT),
// several files at once when multiple is set: Gio's explorer can't on
// Android.
// OpenFiles shows the system's document picker.
func OpenFiles(h platform.Host, exts []string, multiple bool) ([]io.ReadCloser, error) {
	files, err := pickDocuments(context.Background(), h, extMIMEs(exts), multiple, 0, false)
	if err != nil {
		return nil, err
	}
	rcs := make([]io.ReadCloser, len(files))
	for i, f := range files {
		rcs[i] = f
	}
	return rcs, nil
}

// pickMedia shows the system Photo Picker where Android has it (no
// permission needed), the document picker for images and videos
// otherwise, and returns the original files.
// PickMedia shows the Photo Picker, or the document picker for images
// and videos without it.
func PickMedia(ctx context.Context, h platform.Host, limit int, photos, videos bool) ([]io.ReadCloser, error) {
	var mimes []string
	if photos {
		mimes = append(mimes, "image/*")
	}
	if videos {
		mimes = append(mimes, "video/*")
	}
	files, err := pickDocuments(ctx, h, mimes, limit != 1, max(limit, 0), true)
	if err != nil {
		return nil, err
	}
	rcs := make([]io.ReadCloser, len(files))
	for i, f := range files {
		rcs[i] = f
	}
	return rcs, nil
}

// pickDocuments runs mobile.PickDocuments and opens what it returns.
func pickDocuments(ctx context.Context, h platform.Host, mimes []string, multiple bool, limit int, media bool) ([]androidFile, error) {
	var ch <-chan mobile.Documents
	err := platform.OnAndroidMain(ctx, h, func(view uintptr) error {
		var err error
		ch, err = mobile.PickDocuments(app.JavaVM(), app.AppContext(), view, mimes, multiple, limit, media)
		return err
	})
	if err != nil {
		return nil, err
	}
	select {
	case ds := <-ch:
		switch {
		case errors.Is(ds.Err, mobile.ErrCanceled):
			return nil, platform.ErrCanceled
		case ds.Err != nil:
			return nil, ds.Err
		}
		files := make([]androidFile, len(ds.Docs))
		for i, d := range ds.Docs {
			files[i] = androidFile{File: os.NewFile(uintptr(d.FD), d.Name), name: d.Name, size: d.Size}
		}
		return files, nil
	case <-ctx.Done():
		// Close what the picker still returns.
		go func() {
			for _, d := range (<-ch).Docs {
				os.NewFile(uintptr(d.FD), d.Name).Close()
			}
		}()
		return nil, ctx.Err()
	}
}

// extMIMEs turns extensions into the MIME types the document picker
// filters by; nil (any file) when one has no known type.
func extMIMEs(exts []string) []string {
	var mimes []string
	seen := map[string]bool{}
	for _, e := range exts {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		t := platform.MIMEType(strings.ToLower(e))
		if t == "" {
			return nil
		}
		t, _, _ = strings.Cut(t, ";")
		if t = strings.TrimSpace(t); !seen[t] {
			seen[t] = true
			mimes = append(mimes, t)
		}
	}
	return mimes
}

// PickPhoto shows the photo picker, or the camera.
func PickPhoto(ctx context.Context, h platform.Host, maxSide int, camera bool) (Photo, error) {
	var ch <-chan mobile.Photo
	err := platform.OnAndroidMain(ctx, h, func(view uintptr) error {
		var err error
		ch, err = mobile.PickPhoto(app.JavaVM(), app.AppContext(), view, maxSide, camera)
		return err
	})
	if err != nil {
		return Photo{}, err
	}
	select {
	case p := <-ch:
		switch {
		case errors.Is(p.Err, mobile.ErrCanceled):
			return Photo{}, platform.ErrCanceled
		case p.Err != nil:
			return Photo{}, p.Err
		}
		return Photo{Name: p.Name, JPEG: p.JPEG}, nil
	case <-ctx.Done():
		return Photo{}, ctx.Err()
	}
}

// PickPhotos shows the photo picker for several photos.
func PickPhotos(ctx context.Context, h platform.Host, maxSide, limit int) ([]Photo, error) {
	var ch <-chan mobile.Photos
	err := platform.OnAndroidMain(ctx, h, func(view uintptr) error {
		var err error
		ch, err = mobile.PickPhotos(app.JavaVM(), app.AppContext(), view, maxSide, limit)
		return err
	})
	if err != nil {
		return nil, err
	}
	select {
	case ps := <-ch:
		switch {
		case errors.Is(ps.Err, mobile.ErrCanceled):
			return nil, platform.ErrCanceled
		case ps.Err != nil:
			return nil, ps.Err
		}
		files := make([]Photo, len(ps.Photos))
		for i, p := range ps.Photos {
			files[i] = Photo{Name: p.Name, JPEG: p.JPEG}
		}
		return files, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// SaveToPhotos adds the file to Pictures/<album> (Movies for videos)
// through MediaStore.
func SaveToPhotos(ctx context.Context, path, mime, album string) error {
	return mobile.SaveToGallery(app.JavaVM(), app.AppContext(), path, mime, album)
}
