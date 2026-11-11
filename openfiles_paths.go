//go:build windows || darwin

package giode

import (
	"io"
	"os"
	"sync"
)

// fileDialogMu shows one file dialog at a time, as Gio's explorer did;
// openFileDialog holds it.
var fileDialogMu sync.Mutex

// chooseFiles shows the system's open dialog (openFileDialog: the
// Windows one, NSOpenPanel on macOS, the document picker on iOS) and
// opens the chosen files.
func (a *App) chooseFiles(exts []string, multiple bool) ([]io.ReadCloser, error) {
	paths, err := a.openFileDialog(exts, multiple)
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
