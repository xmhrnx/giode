package giode

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"

	"gioui.org/app"

	"github.com/xmhrnx/giode/internal/mobile"
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
func (a *App) chooseFiles(exts []string, multiple bool) ([]io.ReadCloser, error) {
	files, err := a.pickDocuments(context.Background(), extMIMEs(exts), multiple, 0, false)
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
func (a *App) pickMedia(ctx context.Context, opts MediaOptions) ([]PickedFile, error) {
	var mimes []string
	if opts.Photos {
		mimes = append(mimes, "image/*")
	}
	if opts.Videos {
		mimes = append(mimes, "video/*")
	}
	files, err := a.pickDocuments(ctx, mimes, opts.Limit != 1, max(opts.Limit, 0), true)
	if err != nil {
		return nil, err
	}
	picked := make([]PickedFile, len(files))
	for i, f := range files {
		picked[i] = describe(f)
	}
	return picked, nil
}

// pickDocuments runs mobile.PickDocuments and opens what it returns.
func (a *App) pickDocuments(ctx context.Context, mimes []string, multiple bool, limit int, media bool) ([]androidFile, error) {
	var ch <-chan mobile.Documents
	err := a.onAndroidMain(ctx, func(view uintptr) error {
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
			return nil, ErrCanceled
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
		t := typeByExtension(strings.ToLower(e))
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
