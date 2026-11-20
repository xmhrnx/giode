//go:build !windows && !darwin && !android

package picker

import (
	"io"

	"github.com/xmhrnx/giode/internal/platform"
)

// OpenFiles: Linux and the web use Gio's explorer.
func OpenFiles(h platform.Host, exts []string, multiple bool) ([]io.ReadCloser, error) {
	return nil, ErrUseFallback
}
