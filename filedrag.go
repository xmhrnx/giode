package giode

import (
	"sync"

	"gioui.org/f32"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/elements/dnd"
	"github.com/xmhrnx/giode/internal/platform/filedrag"
)

// FileDragKind is the stage of a FileDrag.
type FileDragKind int

const (
	// FileDragEnter: files were dragged into the window.
	FileDragEnter FileDragKind = iota + 1
	// FileDragOver: they moved over it.
	FileDragOver
	// FileDragLeave: they left without being dropped, or the drag was
	// canceled.
	FileDragLeave
	// FileDragDrop: they were dropped; OnFileDrop follows with their
	// paths.
	FileDragDrop
)

// String returns the kind's name.
func (k FileDragKind) String() string {
	switch k {
	case FileDragEnter:
		return "enter"
	case FileDragOver:
		return "over"
	case FileDragLeave:
		return "leave"
	case FileDragDrop:
		return "drop"
	}
	return "none"
}

// FileDrag is files from the file manager dragged over the window,
// for highlighting where they can be dropped.
type FileDrag struct {
	Kind FileDragKind
	// Position is where the pointer is, in window pixels as layout
	// uses them (compare it with a component's bounds).
	Position f32.Point
}

// fileDragState is the drag in progress, read by App.FileDrag during
// layout.
type fileDragState struct {
	mu     sync.Mutex
	active bool
	pos    f32.Point
}

// FileDrag returns the drag of files over the window in progress, if
// any, as of the last FileDrag event: in a view, a drop zone can draw
// itself highlighted while ok is true. The window redraws when it
// changes. It needs OnFileDrop or OnFileDrag set.
func (a *App) FileDrag() (d FileDrag, ok bool) {
	a.drag.mu.Lock()
	defer a.drag.mu.Unlock()
	if !a.drag.active {
		return FileDrag{}, false
	}
	return FileDrag{Kind: FileDragOver, Position: a.drag.pos}, true
}

// fileDragWanted reports whether the app takes file drops.
func (a *App) fileDragWanted() bool {
	return a.cfg.OnFileDrop != nil || a.cfg.OnFileDrag != nil || a.cfg.FileDropTargets
}

// deliverFileDrag records a drag event and hands it to OnFileDrag on
// the UI goroutine. It's safe from any goroutine.
func (a *App) deliverFileDrag(kind FileDragKind, pos f32.Point) {
	a.drag.mu.Lock()
	changed := a.drag.active != (kind == FileDragEnter || kind == FileDragOver) || a.drag.pos != pos
	a.drag.active = kind == FileDragEnter || kind == FileDragOver
	a.drag.pos = pos
	a.drag.mu.Unlock()
	if kind == FileDragOver && !changed {
		return
	}
	a.Post(func() {
		// The view's drop targets follow it (dnd.Files); the drop
		// itself goes to them in deliverFileDrop.
		switch kind {
		case FileDragEnter:
			dnd.External(dnd.ExternalEnter, pos, nil)
		case FileDragOver:
			dnd.External(dnd.ExternalMove, pos, nil)
		case FileDragLeave:
			dnd.External(dnd.ExternalLeave, pos, nil)
		}
		if a.cfg.OnFileDrag != nil {
			a.cfg.OnFileDrag(FileDrag{Kind: kind, Position: pos})
		}
		if kind != FileDragOver {
			// Memoized views reading FileDrag redraw.
			elements.InvalidateCaches()
		}
	})
}

// deliverFileDrop ends a drag with a drop: FileDragDrop, then
// OnFileDrop with the paths. It's safe from any goroutine.
// A drop target accepting DroppedFiles under the pointer takes the
// files instead of OnFileDrop.
func (a *App) deliverFileDrop(paths []string, pos f32.Point) {
	a.deliverFileDrag(FileDragDrop, pos)
	a.Post(func() {
		if dnd.External(dnd.ExternalDrop, pos, paths) || len(paths) == 0 {
			return
		}
		if a.cfg.OnFileDrop != nil {
			a.cfg.OnFileDrop(paths)
		}
	})
}

func init() {
	filedrag.Handler.Drag = func(kind filedrag.Kind, x, y float32) {
		if a := currentApp.Load(); a != nil {
			a.deliverFileDrag(FileDragKind(kind), f32.Pt(x, y))
		}
	}
	filedrag.Handler.Drop = func(paths []string, x, y float32) {
		if a := currentApp.Load(); a != nil {
			a.deliverFileDrop(paths, f32.Pt(x, y))
		}
	}
}
