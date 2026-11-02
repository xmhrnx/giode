package icon

import (
	"errors"
	"fmt"
	"image"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"
	"sync"

	"gioui.org/layout"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
)

// Source is anything that renders as an icon: a *Document, a Path, an
// *IconVGIcon, or a custom type.
type Source interface {
	Icon(st ...styles.Styles) elements.Element
}

// Path is a Source from raw SVG path data, for icon sets distributed
// as path strings rather than files.
type Path struct {
	// Data is the SVG path data.
	Data string
	// ViewBox is the coordinate space of Data; zero means 24x24.
	ViewBox image.Point
	// StrokeWidth, when positive, strokes the path instead of filling
	// it, in viewBox units.
	StrokeWidth float32
}

// Icon renders the path.
func (p Path) Icon(st ...styles.Styles) elements.Element {
	vb := p.ViewBox
	if vb == (image.Point{}) {
		vb = image.Pt(24, 24)
	}
	return &svgEl{path: compileCached(p.Data), viewW: float32(vb.X), viewH: float32(vb.Y), st: first(st), stroke: p.StrokeWidth}
}

// Registry is a named collection of icons. Most apps use the package
// level functions (Register, Named, ...), which share a default
// registry; separate registries suit libraries that must not collide
// with an app's names. A Registry is safe for concurrent use.
type Registry struct {
	mu    sync.RWMutex
	icons map[string]Source
	// OnMissing, when set, is called the first time Named is asked for
	// an unknown name.
	OnMissing func(name string)
	missing   map[string]bool
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{icons: map[string]Source{}, missing: map[string]bool{}}
}

// Default is the registry behind the package-level functions.
var Default = NewRegistry()

// Register adds (or replaces) an icon under name.
func (r *Registry) Register(name string, src Source) {
	r.mu.Lock()
	r.icons[name] = src
	delete(r.missing, name)
	r.mu.Unlock()
}

// RegisterSVG parses data and registers it under name.
func (r *Registry) RegisterSVG(name string, data []byte) error {
	d, err := LoadSVG(data)
	if err != nil {
		return fmt.Errorf("icon %q: %w", name, err)
	}
	r.Register(name, d)
	return nil
}

// RegisterFS registers every .svg file under dir in fsys (recursively),
// named by its path relative to dir without the extension:
// "icons/home.svg" becomes "home" and "icons/brand/github.svg"
// "brand/github" for dir "icons". Use "." for the root of fsys. It
// registers every file it can and returns the errors of those it
// couldn't, joined.
func (r *Registry) RegisterFS(fsys fs.FS, dir string) error {
	var errs []error
	walkErr := fs.WalkDir(fsys, dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			errs = append(errs, err)
			return nil
		}
		if d.IsDir() || !strings.EqualFold(path.Ext(p), ".svg") {
			return nil
		}
		rel := strings.TrimPrefix(p, dir)
		rel = strings.TrimPrefix(rel, "/")
		if dir == "." {
			rel = p
		}
		name := strings.TrimSuffix(rel, path.Ext(rel))
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			errs = append(errs, err)
			return nil
		}
		if err := r.RegisterSVG(name, data); err != nil {
			errs = append(errs, err)
		}
		return nil
	})
	if walkErr != nil {
		errs = append(errs, walkErr)
	}
	return errors.Join(errs...)
}

// RegisterDir registers every .svg file under a directory on disk; see
// RegisterFS.
func (r *Registry) RegisterDir(dir string) error {
	return r.RegisterFS(os.DirFS(dir), ".")
}

// Lookup returns the icon registered under name.
func (r *Registry) Lookup(name string) (Source, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	src, ok := r.icons[name]
	return src, ok
}

// Unregister removes an icon.
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	delete(r.icons, name)
	r.mu.Unlock()
}

// Names returns the registered names, sorted.
func (r *Registry) Names() []string {
	r.mu.RLock()
	names := make([]string, 0, len(r.icons))
	for n := range r.icons {
		names = append(names, n)
	}
	r.mu.RUnlock()
	sort.Strings(names)
	return names
}

// Named renders the icon registered under name. An unknown name
// renders as empty space of the icon's size - never a panic in the
// middle of a frame - and is reported once through OnMissing.
func (r *Registry) Named(name string, st ...styles.Styles) elements.Element {
	if src, ok := r.Lookup(name); ok {
		return src.Icon(st...)
	}
	r.mu.Lock()
	report := !r.missing[name]
	r.missing[name] = true
	onMissing := r.OnMissing
	r.mu.Unlock()
	if report && onMissing != nil {
		onMissing(name)
	}
	return &missingEl{st: first(st)}
}

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
