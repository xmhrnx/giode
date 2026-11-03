//go:build !windows

package giode

import "io"

// chooseFiles shows the platform's open dialog through Gio's explorer.
func (a *App) chooseFiles(exts []string, multiple bool) ([]io.ReadCloser, error) {
	if !multiple {
		f, err := a.explorer.ChooseFile(exts...)
		if err != nil {
			return nil, err
		}
		return []io.ReadCloser{f}, nil
	}
	return a.explorer.ChooseFiles(exts...)
}
