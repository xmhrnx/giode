package giode

import (
	"bytes"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"

	"gioui.org/app"
	"github.com/xmhrnx/giode/internal/platform/filedrag"
	"github.com/xmhrnx/giode/internal/platform/window"
)

// decodeIcon parses PNG or JPEG icon bytes into an image.
func decodeIcon(data []byte) (image.Image, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	return img, err
}

// setIconBytes stores the window icon. The icon is applied to the
// window once its platform handle becomes available.
func (a *App) setIconBytes(data []byte) {
	if len(data) == 0 {
		return
	}
	img, err := decodeIcon(data)
	if err != nil {
		a.logf(LogLevelWarning, "giode: invalid window icon: %v", err)
		os.Stderr.WriteString("giode: invalid window icon: " + err.Error() + "\n")
		return
	}
	a.icon = img
}

// platformView handles the native window once it exists: the icon
// (Windows; elsewhere it comes from the application bundle), the
// window procedure (Windows) and file drops.
func (a *App) platformView(ev app.ViewEvent) {
	if !ev.Valid() {
		return
	}
	if a.icon != nil && !a.iconApplied && window.SetIcon(ev, a.icon) {
		a.iconApplied = true
	}
	window.Subclass(a.host(), ev, a.subclassOptions())
	if a.fileDragWanted() {
		if why := filedrag.ViewReady(ev); why != "" {
			a.warnNoFileDrop(why)
		}
	}
}

// subclassOptions are the drops the window takes, for Windows' window
// procedure.
func (a *App) subclassOptions() window.SubclassOptions {
	return window.SubclassOptions{
		AcceptFiles: a.fileDragWanted(),
		DragEvents:  a.cfg.OnFileDrag != nil || a.cfg.FileDropTargets,
	}
}

// warnNoFileDrop logs, once, that file drops won't arrive.
func (a *App) warnNoFileDrop(why string) {
	if !a.dropWarned {
		a.dropWarned = true
		a.logf(LogLevelWarning, "giode: Config.OnFileDrop and OnFileDrag %s", why)
	}
}
