//go:build windows || (darwin && !ios) || ios

package picker

import (
	"io"
	"os"

	"github.com/xmhrnx/giode/internal/platform"
)

// chooseFiles shows the system's open dialog (openFileDialog: the
// Windows one, NSOpenPanel on macOS, the document picker on iOS) and
// opens the chosen files.
// OpenFiles shows the open dialog and opens the files chosen.
func OpenFiles(h platform.Host, exts []string, multiple bool) ([]io.ReadCloser, error) {
	paths, err := openFileDialog(h, exts, multiple)
	if err != nil {
		return nil, err
	}
	files := make([]io.ReadCloser, 0, len(paths))
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			for _, f := range files {
				f.Close()
			}
			return nil, err
		}
		files = append(files, f)
	}
	return files, nil
}
