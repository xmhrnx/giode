// Package filedrag carries file drags between the app's window and the
// system: files dragged in from the file manager, and Draggable.Export
// dragging out.
package filedrag

import (
	"net/url"
	"strings"
)

// Kind is the stage of a drag into the window, as giode.FileDragKind.
type Kind int

const (
	Enter Kind = iota + 1
	Over
	Leave
)

// Handler receives drags into the window, in window pixels.
var Handler struct {
	Drag func(kind Kind, x, y float32)
	Drop func(paths []string, x, y float32)
}

func drag(kind Kind, x, y float32) {
	if f := Handler.Drag; f != nil {
		f(kind, x, y)
	}
}

func drop(paths []string, x, y float32) {
	if f := Handler.Drop; f != nil {
		f(paths, x, y)
	}
}

// ParseURIList reads the local paths of a text/uri-list, the format
// of X11 and Wayland drops.
func ParseURIList(list string) []string {
	var paths []string
	for _, line := range strings.Split(list, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		u, err := url.Parse(line)
		if err != nil || u.Scheme != "file" || (u.Host != "" && u.Host != "localhost") {
			continue
		}
		if u.Path != "" {
			paths = append(paths, u.Path)
		}
	}
	return paths
}
