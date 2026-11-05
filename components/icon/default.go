package icon

import (
	"io/fs"

	"gioui.org/layout"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
)

type missingEl struct{ st styles.Styles }

func (m *missingEl) Layout(gtx layout.Context) layout.Dimensions {
	size, _ := iconSize(gtx, m.st)
	return layout.Dimensions{Size: size}
}

// Register adds (or replaces) an icon in the default registry.
func Register(name string, src Source) { Default.Register(name, src) }

// RegisterSVG parses data and registers it in the default registry.
func RegisterSVG(name string, data []byte) error { return Default.RegisterSVG(name, data) }

// RegisterFS registers every .svg file under dir in fsys with the
// default registry; see Registry.RegisterFS.
func RegisterFS(fsys fs.FS, dir string) error { return Default.RegisterFS(fsys, dir) }

// MustRegisterFS is RegisterFS, panicking on error - convenient for an
// embed.FS known at compile time.
func MustRegisterFS(fsys fs.FS, dir string) {
	if err := RegisterFS(fsys, dir); err != nil {
		panic(err)
	}
}

// RegisterDir registers every .svg file under a directory on disk with
// the default registry.
func RegisterDir(dir string) error { return Default.RegisterDir(dir) }

// Lookup returns an icon from the default registry.
func Lookup(name string) (Source, bool) { return Default.Lookup(name) }

// Names returns the names in the default registry, sorted.
func Names() []string { return Default.Names() }

// Named renders an icon from the default registry; see
// Registry.Named.
func Named(name string, st ...styles.Styles) elements.Element { return Default.Named(name, st...) }
