package giode

import (
	"github.com/xmhrnx/giode/host"

	"context"
	"errors"
	"io"
	"strings"

	"gioui.org/x/explorer"
)

// ErrCanceled is returned by OpenFile, OpenFiles and SaveFile when the
// user closes the dialog without choosing a file.
var ErrCanceled = errors.New("giode: canceled by the user")

// ErrNotSupported is returned by features the current platform lacks.
var ErrNotSupported = host.ErrNotSupported

// FileFilter limits the files OpenFile and OpenFiles offer.
type FileFilter struct {
	// Extensions lists the accepted file extensions, with or without
	// the leading dot: "wav", ".mp3". Empty accepts any file.
	Extensions []string
}

func extensions(filters []FileFilter) []string {
	var exts []string
	for _, f := range filters {
		for _, e := range f.Extensions {
			if e == "" {
				continue
			}
			if !strings.HasPrefix(e, ".") {
				e = "." + e
			}
			exts = append(exts, e)
		}
	}
	return exts
}

// OpenFile shows the platform's file picker and returns the file the
// user chose, open for reading. On Android and iOS that's the system
// picker, the only way apps there can read the user's files; on
// desktops it's the native open dialog (through the XDG desktop portal
// on Linux).
//
// OpenFile blocks until the user decides, so call it from a goroutine,
// never from a click handler or anything else running on the UI
// goroutine - the dialog needs the event loop running:
//
//	go func() {
//		f, err := app.OpenFile(ctx, giode.FileFilter{Extensions: []string{"wav", "mp3"}})
//		if errors.Is(err, giode.ErrCanceled) {
//			return
//		}
//		...
//	}()
//
// It returns ErrCanceled when the user closes the picker, and
// ctx.Err() when ctx ends first (the picker may stay open; a file
// chosen after that is closed). One file dialog shows at a time;
// further calls wait for it.
func (a *App) OpenFile(ctx context.Context, filter ...FileFilter) (io.ReadCloser, error) {
	exts := extensions(filter)
	return await(ctx, func() (io.ReadCloser, error) {
		fs, err := a.chooseFiles(exts, false)
		if err != nil {
			return nil, err
		}
		return fs[0], nil
	}, closeFile[io.ReadCloser])
}

// OpenFiles is OpenFile for choosing several files at once.
func (a *App) OpenFiles(ctx context.Context, filter ...FileFilter) ([]io.ReadCloser, error) {
	exts := extensions(filter)
	return await(ctx, func() ([]io.ReadCloser, error) {
		return a.chooseFiles(exts, true)
	}, func(fs []io.ReadCloser) {
		for _, f := range fs {
			f.Close()
		}
	})
}

// SaveFile shows the platform's save dialog, suggesting name, and
// returns the file the user chose, open for writing; close it when
// done. On Android that's the system's "create document" picker. Like
// OpenFile, it blocks until the user decides: call it from a goroutine.
func (a *App) SaveFile(ctx context.Context, name string) (io.WriteCloser, error) {
	return await(ctx, func() (io.WriteCloser, error) {
		return a.explorer.CreateFile(name)
	}, closeFile[io.WriteCloser])
}

func closeFile[T io.Closer](f T) { f.Close() }

// await runs f on its own goroutine and returns its result, or
// ctx.Err() if ctx ends first, in which case a result arriving later
// is passed to discard (when f succeeded). Explorer errors map to
// giode's.
func await[T any](ctx context.Context, f func() (T, error), discard func(T)) (T, error) {
	type result struct {
		v   T
		err error
	}
	done := make(chan result, 1)
	go func() {
		v, err := f()
		done <- result{v, err}
	}()
	select {
	case r := <-done:
		switch {
		case errors.Is(r.err, explorer.ErrUserDecline):
			r.err = ErrCanceled
		case errors.Is(r.err, explorer.ErrNotAvailable):
			r.err = ErrNotSupported
		}
		return r.v, r.err
	case <-ctx.Done():
		go func() {
			if r := <-done; r.err == nil {
				discard(r.v)
			}
		}()
		var zero T
		return zero, ctx.Err()
	}
}

// splitPaths splits paths that each end with a NUL, the one byte a
// path can't hold, leaving out empty ones.
func splitPaths(b []byte) []string {
	var out []string
	for _, p := range strings.Split(string(b), "\x00") {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
