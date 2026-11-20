// Package picker shows the system's pickers: files, folders, photos
// and media.
package picker

import (
	"errors"
	"strings"
	"sync"
)

// ErrUseFallback means the platform has no picker of its own for the
// request: the caller uses the portable one (the file picker, Gio's
// explorer).
var ErrUseFallback = errors.New("giode: no native picker")

// Photo is a picked photo, as an upright JPEG.
type Photo struct {
	Name string
	JPEG []byte
}

// dialogMu shows one file dialog at a time, as Gio's explorer did.
var dialogMu sync.Mutex

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
