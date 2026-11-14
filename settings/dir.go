//go:build !js

package settings

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gioui.org/app"
)

// dataBackend is the file in the app's data directory, found on first
// use: on Android the directory is known only once the app runs.
type dataBackend struct {
	once sync.Once
	file fileBackend
	err  error
}

func defaultBackend() backend { return &dataBackend{} }

func (d *dataBackend) resolve() error {
	d.once.Do(func() {
		dir, err := app.DataDir()
		if err != nil {
			d.err = err
			return
		}
		d.file = fileBackend{path: filepath.Join(dir, appName(), "settings.json")}
	})
	return d.err
}

func (d *dataBackend) load() ([]byte, error) {
	if err := d.resolve(); err != nil {
		return nil, err
	}
	return d.file.load()
}

func (d *dataBackend) save(b []byte) error {
	if err := d.resolve(); err != nil {
		return err
	}
	return d.file.save(b)
}

// appName names the app's folder: its ID (giode build sets it), else
// the executable's name.
func appName() string {
	if app.ID != "" {
		return app.ID
	}
	exe, _ := os.Executable()
	return strings.TrimSuffix(filepath.Base(exe), filepath.Ext(exe))
}
